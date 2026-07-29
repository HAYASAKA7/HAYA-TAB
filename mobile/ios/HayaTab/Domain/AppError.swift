import Foundation

enum AppError: Error, Equatable, Sendable {
    case authentication
    case transport(String)
    case malformedManifest
    case unsafeRemotePath(String)
    case unsupportedDocument(String)
    case localStorage(String)
    case downloadIntegrity
    case remoteChanged

    static var presentationFixtures: [AppError] {
        [
            .authentication,
            .transport("private transport detail"),
            .malformedManifest,
            .unsafeRemotePath("private remote path"),
            .unsupportedDocument("private document name"),
            .localStorage("private storage detail"),
            .downloadIntegrity,
            .remoteChanged,
        ]
    }

    var code: String {
        switch self {
        case .authentication: "authentication"
        case .transport: "transport"
        case .malformedManifest: "malformed_manifest"
        case .unsafeRemotePath: "unsafe_remote_path"
        case .unsupportedDocument: "unsupported_document"
        case .localStorage: "local_storage"
        case .downloadIntegrity: "download_integrity"
        case .remoteChanged: "remote_changed"
        }
    }

    var title: String {
        switch self {
        case .authentication:
            "Couldn’t sign in"
        case .transport:
            "Cloud connection failed"
        case .malformedManifest:
            "Library data is invalid"
        case .unsafeRemotePath:
            "Unsafe cloud path"
        case .unsupportedDocument:
            "Unsupported document"
        case .localStorage:
            "Couldn’t save on this device"
        case .downloadIntegrity:
            "Download couldn’t be verified"
        case .remoteChanged:
            "Cloud document changed"
        }
    }

    var recoverySuggestion: String {
        switch self {
        case .authentication:
            "Check the server address, username, and password, then try again."
        case .transport:
            "Check your connection and try again. Your existing offline files are still available."
        case .malformedManifest:
            "Refresh after the desktop library has completed its next cloud sync."
        case .unsafeRemotePath:
            "Remove the invalid item from the desktop library and sync again."
        case .unsupportedDocument:
            "Open a PDF or supported Guitar Pro document."
        case .localStorage:
            "Free some device storage and try again."
        case .downloadIntegrity:
            "Try the download again. The previous offline copy was not changed."
        case .remoteChanged:
            "Refresh the library, then download the latest version."
        }
    }

    var isRetryable: Bool {
        switch self {
        case .authentication, .transport, .localStorage, .downloadIntegrity, .remoteChanged:
            true
        case .malformedManifest, .unsafeRemotePath, .unsupportedDocument:
            false
        }
    }
}
