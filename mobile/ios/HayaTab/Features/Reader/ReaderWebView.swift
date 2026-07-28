import Foundation
import Observation
import SwiftUI
import WebKit

@MainActor
@Observable
final class ReaderSession {
    private(set) var error: AppError?
    private(set) var isLoaded = false
    @ObservationIgnored private weak var webView: WKWebView?
    @ObservationIgnored private var documentURL: URL?

    func attach(_ webView: WKWebView, documentURL: URL) {
        self.webView = webView
        self.documentURL = documentURL
        error = nil
        isLoaded = false
    }

    func handle(_ event: ReaderEvent) {
        switch event {
        case .ready:
            loadDocument()
        case .loaded:
            isLoaded = true
        case .failed:
            error = .unsupportedDocument(
                documentURL?.lastPathComponent ?? "document")
        }
    }

    func send(_ command: ReaderCommand) {
        send(command, maximumDocumentBytes: 0)
    }

    private func loadDocument() {
        guard let documentURL else {
            error = .localStorage("The offline document is unavailable.")
            return
        }
        do {
            let values = try documentURL.resourceValues(
                forKeys: [.fileSizeKey, .isRegularFileKey, .isSymbolicLinkKey])
            let byteCount = values.fileSize ?? 0
            guard values.isRegularFile == true,
                  values.isSymbolicLink != true,
                  byteCount > 0,
                  byteCount <= 128 * 1024 * 1024 else {
                throw AppError.downloadIntegrity
            }
            let data = try Data(contentsOf: documentURL, options: [.mappedIfSafe])
            send(.load(data), maximumDocumentBytes: byteCount)
        } catch let appError as AppError {
            error = appError
        } catch {
            self.error = .localStorage("The offline document could not be read.")
        }
    }

    private func send(
        _ command: ReaderCommand,
        maximumDocumentBytes: Int
    ) {
        guard let webView else {
            error = .localStorage("The document viewer is unavailable.")
            return
        }
        do {
            let object = try ReaderBridgeContract.commandObject(
                command,
                maximumDocumentBytes: maximumDocumentBytes)
            Task { @MainActor in
                do {
                    _ = try await webView.callAsyncJavaScript(
                        "return await window.hayaTabViewer.receive(command)",
                        arguments: ["command": object],
                        in: nil,
                        contentWorld: .page)
                } catch {
                    self.error = .unsupportedDocument(
                        self.documentURL?.lastPathComponent ?? "document")
                }
            }
        } catch let appError as AppError {
            error = appError
        } catch {
            self.error = .downloadIntegrity
        }
    }
}

struct ReaderWebView: UIViewRepresentable {
    let documentURL: URL
    let kind: DocumentKind
    let session: ReaderSession

    func makeCoordinator() -> Coordinator {
        Coordinator(session: session)
    }

    func makeUIView(context: Context) -> WKWebView {
        let configuration = WKWebViewConfiguration()
        configuration.websiteDataStore = .nonPersistent()
        configuration.defaultWebpagePreferences.allowsContentJavaScript = true
        configuration.allowsInlineMediaPlayback = true

        if kind == .guitarPro {
            configuration.userContentController.addScriptMessageHandler(
                context.coordinator,
                contentWorld: .page,
                name: Coordinator.bridgeName)
        }

        let webView = WKWebView(frame: .zero, configuration: configuration)
        webView.isOpaque = false
        webView.backgroundColor = .clear
        webView.scrollView.backgroundColor = .clear
        webView.navigationDelegate = context.coordinator

        switch kind {
        case .pdf:
            webView.loadFileURL(
                documentURL,
                allowingReadAccessTo: documentURL.deletingLastPathComponent())
        case .guitarPro:
            session.attach(webView, documentURL: documentURL)
            guard let viewerURL = Bundle.main.url(
                forResource: "index",
                withExtension: "html",
                subdirectory: "Viewer") else {
                session.handle(.failed("viewer_missing"))
                return webView
            }
            context.coordinator.allowedViewerRoot =
                viewerURL.deletingLastPathComponent().standardizedFileURL
            webView.loadFileURL(
                viewerURL,
                allowingReadAccessTo: viewerURL.deletingLastPathComponent())
        }
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {}

    static func dismantleUIView(_ webView: WKWebView, coordinator: Coordinator) {
        webView.stopLoading()
        webView.navigationDelegate = nil
        webView.configuration.userContentController.removeScriptMessageHandler(
            forName: Coordinator.bridgeName,
            contentWorld: .page)
    }

    @MainActor
    final class Coordinator: NSObject,
        WKNavigationDelegate,
        WKScriptMessageHandlerWithReply
    {
        static let bridgeName = "hayaBridge"
        let session: ReaderSession
        var allowedViewerRoot: URL?

        init(session: ReaderSession) {
            self.session = session
        }

        func userContentController(
            _ userContentController: WKUserContentController,
            didReceive message: WKScriptMessage,
            replyHandler: @escaping @MainActor @Sendable
                (Any?, String?) -> Void
        ) {
            guard message.name == Self.bridgeName else {
                replyHandler(nil, "unknown_bridge")
                return
            }
            do {
                let event = try ReaderBridgeContract.event(from: message.body)
                session.handle(event)
                replyHandler(["ok": true], nil)
            } catch let error as ReaderBridgeError {
                replyHandler(
                    ["ok": false, "error": ["code": bridgeCode(for: error)]],
                    nil)
            } catch {
                replyHandler(
                    ["ok": false, "error": ["code": "invalid_payload"]],
                    nil)
            }
        }

        func webView(
            _ webView: WKWebView,
            decidePolicyFor navigationAction: WKNavigationAction,
            decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
        ) {
            guard let url = navigationAction.request.url,
                  url.isFileURL else {
                decisionHandler(.cancel)
                return
            }
            if let allowedViewerRoot {
                let candidate = url.standardizedFileURL
                let allowed = candidate == allowedViewerRoot
                    || candidate.path.hasPrefix(allowedViewerRoot.path + "/")
                decisionHandler(allowed ? .allow : .cancel)
            } else {
                decisionHandler(.allow)
            }
        }

        private func bridgeCode(for error: ReaderBridgeError) -> String {
            switch error {
            case .unsupportedVersion: "unsupported_version"
            case .unknownMessage: "unknown_message"
            case .invalidPayload: "invalid_payload"
            case .payloadTooLarge: "payload_too_large"
            }
        }
    }
}
