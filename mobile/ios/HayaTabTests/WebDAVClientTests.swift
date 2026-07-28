import Foundation
import XCTest
@testable import HayaTab

final class WebDAVClientTests: XCTestCase {
    override func tearDown() {
        URLProtocolStub.handler = nil
        super.tearDown()
    }

    func testGetAddsBasicAuthenticationOnConfiguredOrigin() async throws {
        URLProtocolStub.handler = { request in
            XCTAssertEqual(request.url?.absoluteString, "https://cloud.example.test/music/scores/primer.pdf")
            XCTAssertEqual(
                request.value(forHTTPHeaderField: "Authorization"),
                "Basic \(Data("haya:test-only-secret".utf8).base64EncodedString())")
            return Self.response(for: request, status: 200, data: Data("document".utf8))
        }

        let data = try await makeClient().get(relativePath: "scores/primer.pdf")

        XCTAssertEqual(data, Data("document".utf8))
    }

    func testPropfindUsesDepthZero() async throws {
        URLProtocolStub.handler = { request in
            XCTAssertEqual(request.httpMethod, "PROPFIND")
            XCTAssertEqual(request.value(forHTTPHeaderField: "Depth"), "0")
            return Self.response(for: request, status: 207)
        }

        _ = try await makeClient().propfind(relativePath: "fingerprints/bucket_00.json")
    }

    func testAuthenticationResponseMapsToTypedError() async throws {
        URLProtocolStub.handler = { request in
            Self.response(for: request, status: 401)
        }

        await assertWebDAVError(.authenticationRequired) {
            _ = try await self.makeClient().get(relativePath: "scores/primer.pdf")
        }
    }

    func testMissingResponseMapsToTypedError() async throws {
        URLProtocolStub.handler = { request in
            Self.response(for: request, status: 404)
        }

        await assertWebDAVError(.remoteNotFound) {
            _ = try await self.makeClient().get(relativePath: "scores/missing.pdf")
        }
    }

    func testRejectsTraversalAndForeignOriginsBeforeSending() async throws {
        URLProtocolStub.handler = { _ in
            XCTFail("Rejected paths must not reach URLSession")
            throw URLError(.badURL)
        }
        let client = try makeClient()

        await assertWebDAVError(.invalidRemotePath) {
            _ = try await client.get(relativePath: "scores/../private.pdf")
        }
        await assertWebDAVError(.invalidRemotePath) {
            _ = try await client.get(relativePath: "https://evil.example.test/private.pdf")
        }
    }

    func testRejectsNonHTTPSBaseURL() throws {
        let credential = WebDAVCredential(
            baseURL: try XCTUnwrap(URL(string: "http://cloud.example.test/music/")),
            username: "haya",
            password: "test-only-secret")

        XCTAssertThrowsError(try WebDAVClient(credential: credential, session: makeSession())) { error in
            XCTAssertEqual(error as? WebDAVError, .insecureTransport)
        }
    }

    func testCancellationRemainsCancellationError() async throws {
        URLProtocolStub.handler = { _ in
            throw URLError(.cancelled)
        }

        do {
            _ = try await makeClient().get(relativePath: "scores/primer.pdf")
            XCTFail("Expected cancellation")
        } catch is CancellationError {
            // Expected: callers can distinguish cancellation from a transport failure.
        } catch {
            XCTFail("Expected CancellationError, got \(type(of: error))")
        }
    }

    private func makeClient() throws -> WebDAVClient {
        let credential = WebDAVCredential(
            baseURL: try XCTUnwrap(URL(string: "https://cloud.example.test/music/")),
            username: "haya",
            password: "test-only-secret")
        return try WebDAVClient(credential: credential, session: makeSession())
    }

    private func makeSession() -> URLSession {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [URLProtocolStub.self]
        return URLSession(configuration: configuration)
    }

    private func assertWebDAVError(
        _ expected: WebDAVError,
        operation: () async throws -> Void
    ) async {
        do {
            try await operation()
            XCTFail("Expected \(expected)")
        } catch let error as WebDAVError {
            XCTAssertEqual(error, expected)
        } catch {
            XCTFail("Expected WebDAVError, got \(type(of: error))")
        }
    }

    private static func response(
        for request: URLRequest,
        status: Int,
        data: Data = Data()
    ) -> (HTTPURLResponse, Data) {
        let response = HTTPURLResponse(
            url: request.url!,
            statusCode: status,
            httpVersion: "HTTP/1.1",
            headerFields: nil)!
        return (response, data)
    }
}

private final class URLProtocolStub: URLProtocol, @unchecked Sendable {
    nonisolated(unsafe) static var handler:
        (@Sendable (URLRequest) throws -> (HTTPURLResponse, Data))?

    override class func canInit(with request: URLRequest) -> Bool { true }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest {
        request
    }

    override func startLoading() {
        do {
            let (response, data) = try Self.handler!(request)
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {}
}
