import Foundation

struct WebDAVResponse: Sendable {
    let statusCode: Int
    let data: Data
    let etag: String?
}

protocol WebDAVServing: Sendable {
    func get(path: String) async throws -> WebDAVResponse
    func testConnection() async throws
}

protocol WebDAVClientBuilding: Sendable {
    func makeClient(credential: WebDAVCredential) throws -> any WebDAVServing
}

enum WebDAVError: Error, Equatable, Sendable {
    case authenticationRequired
    case remoteNotFound
    case invalidRemotePath
    case insecureTransport
    case invalidResponse
    case httpStatus(Int)
    case transport
}

struct WebDAVClientFactory: WebDAVClientBuilding, Sendable {
    func makeClient(credential: WebDAVCredential) throws -> any WebDAVServing {
        try WebDAVClient(credential: credential)
    }
}

struct WebDAVClient: WebDAVServing, Sendable {
    private let baseURL: URL
    private let origin: WebOrigin
    private let authorization: String
    private let session: URLSession

    init(credential: WebDAVCredential) throws {
        try self.init(
            credential: credential,
            session: URLSession(
                configuration: .ephemeral,
                delegate: RejectRedirectDelegate(),
                delegateQueue: nil))
    }

    init(credential: WebDAVCredential, session: URLSession) throws {
        let normalizedURL = try Self.normalizedBaseURL(credential.baseURL)
        guard let origin = WebOrigin(url: normalizedURL) else {
            throw WebDAVError.invalidRemotePath
        }

        self.baseURL = normalizedURL
        self.origin = origin
        authorization = "Basic \(Data("\(credential.username):\(credential.password)".utf8).base64EncodedString())"
        self.session = session
    }

    func propfind(relativePath: String) async throws -> Data {
        var request = try makeRequest(method: "PROPFIND", relativePath: relativePath)
        request.setValue("0", forHTTPHeaderField: "Depth")
        return try await perform(request)
    }

    func get(relativePath: String) async throws -> Data {
        try await perform(makeRequest(method: "GET", relativePath: relativePath))
    }

    func get(path: String) async throws -> WebDAVResponse {
        try await performResponse(makeRequest(method: "GET", relativePath: path))
    }

    func testConnection() async throws {
        var request = try makeRequest(method: "PROPFIND", relativePath: "")
        request.setValue("0", forHTTPHeaderField: "Depth")
        _ = try await performResponse(request)
    }

    func download(relativePath: String) async throws -> URL {
        let request = try makeRequest(method: "GET", relativePath: relativePath)
        do {
            let (temporaryURL, response) = try await session.download(for: request)
            _ = try validate(response)
            return temporaryURL
        } catch is CancellationError {
            throw CancellationError()
        } catch let error as URLError where error.code == .cancelled {
            throw CancellationError()
        } catch let error as WebDAVError {
            throw error
        } catch {
            throw WebDAVError.transport
        }
    }

    private func perform(_ request: URLRequest) async throws -> Data {
        try await performResponse(request).data
    }

    private func performResponse(_ request: URLRequest) async throws -> WebDAVResponse {
        do {
            let (data, response) = try await session.data(for: request)
            let response = try validate(response)
            return WebDAVResponse(
                statusCode: response.statusCode,
                data: data,
                etag: response.value(forHTTPHeaderField: "ETag"))
        } catch is CancellationError {
            throw CancellationError()
        } catch let error as URLError where error.code == .cancelled {
            throw CancellationError()
        } catch let error as WebDAVError {
            throw error
        } catch {
            throw WebDAVError.transport
        }
    }

    private func makeRequest(method: String, relativePath: String) throws -> URLRequest {
        let url = try remoteURL(relativePath: relativePath)
        guard WebOrigin(url: url) == origin else {
            throw WebDAVError.invalidRemotePath
        }

        var request = URLRequest(url: url)
        request.httpMethod = method
        request.cachePolicy = .reloadIgnoringLocalCacheData
        request.setValue(authorization, forHTTPHeaderField: "Authorization")
        request.setValue("HAYA-TAB iOS", forHTTPHeaderField: "User-Agent")
        return request
    }

    private func remoteURL(relativePath: String) throws -> URL {
        if relativePath.isEmpty {
            return baseURL
        }
        guard !relativePath.hasPrefix("/"),
              !relativePath.hasPrefix("//"),
              !relativePath.contains("\\"),
              !relativePath.contains("?"),
              !relativePath.contains("#"),
              URLComponents(string: relativePath)?.scheme == nil else {
            throw WebDAVError.invalidRemotePath
        }

        var encodedSegments: [String] = []
        var allowed = CharacterSet.urlPathAllowed
        allowed.remove(charactersIn: "/?#\\")

        for rawSegment in relativePath.split(separator: "/", omittingEmptySubsequences: false) {
            let source = String(rawSegment)
            guard !source.isEmpty,
                  let decoded = source.removingPercentEncoding,
                  decoded != ".",
                  decoded != "..",
                  !decoded.contains("/"),
                  !decoded.contains("\\"),
                  let encoded = decoded.addingPercentEncoding(withAllowedCharacters: allowed) else {
                throw WebDAVError.invalidRemotePath
            }
            encodedSegments.append(encoded)
        }

        guard var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false) else {
            throw WebDAVError.invalidRemotePath
        }
        components.percentEncodedPath += encodedSegments.joined(separator: "/")
        guard let url = components.url else {
            throw WebDAVError.invalidRemotePath
        }
        return url
    }

    private func validate(_ response: URLResponse) throws -> HTTPURLResponse {
        guard let response = response as? HTTPURLResponse,
              let responseURL = response.url,
              WebOrigin(url: responseURL) == origin else {
            throw WebDAVError.invalidResponse
        }

        switch response.statusCode {
        case 200 ..< 300:
            return response
        case 401:
            throw WebDAVError.authenticationRequired
        case 404:
            throw WebDAVError.remoteNotFound
        default:
            throw WebDAVError.httpStatus(response.statusCode)
        }
    }

    private static func normalizedBaseURL(_ url: URL) throws -> URL {
        guard var components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              components.scheme?.lowercased() == "https",
              components.host?.isEmpty == false,
              components.user == nil,
              components.password == nil,
              components.query == nil,
              components.fragment == nil else {
            throw WebDAVError.insecureTransport
        }

        components.scheme = "https"
        components.host = components.host?.lowercased()
        if !components.percentEncodedPath.hasSuffix("/") {
            components.percentEncodedPath += "/"
        }
        guard let normalized = components.url else {
            throw WebDAVError.invalidRemotePath
        }
        return normalized
    }
}

private struct WebOrigin: Equatable, Sendable {
    let scheme: String
    let host: String
    let port: Int

    init?(url: URL) {
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              let scheme = components.scheme?.lowercased(),
              let host = components.host?.lowercased() else {
            return nil
        }
        self.scheme = scheme
        self.host = host
        port = components.port ?? (scheme == "https" ? 443 : 80)
    }
}

private final class RejectRedirectDelegate: NSObject, URLSessionTaskDelegate, @unchecked Sendable {
    func urlSession(
        _ session: URLSession,
        task: URLSessionTask,
        willPerformHTTPRedirection response: HTTPURLResponse,
        newRequest request: URLRequest,
        completionHandler: @escaping @Sendable (URLRequest?) -> Void
    ) {
        completionHandler(nil)
    }
}
