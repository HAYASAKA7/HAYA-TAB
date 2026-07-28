import Foundation
import Security

struct WebDAVCredential: Codable, Sendable {
    let baseURL: URL
    let username: String
    let password: String
}

protocol CredentialStoring: Sendable {
    func load() throws -> WebDAVCredential?
    func save(_ credential: WebDAVCredential) throws
    func delete() throws
}

struct CredentialStore: CredentialStoring, Sendable {
    private let service: String

    init(service: String = "com.hayasaka7.hayatab.webdav") {
        self.service = service
    }

    func load() throws -> WebDAVCredential? {
        var query = baseQuery
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne

        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        if status == errSecItemNotFound {
            return nil
        }
        guard status == errSecSuccess, let data = result as? Data else {
            throw sanitizedError(operation: "read")
        }

        do {
            return try JSONDecoder().decode(WebDAVCredential.self, from: data)
        } catch {
            throw AppError.localStorage("Secure cloud credentials could not be decoded.")
        }
    }

    func save(_ credential: WebDAVCredential) throws {
        let data: Data
        do {
            data = try JSONEncoder().encode(credential)
        } catch {
            throw AppError.localStorage("Secure cloud credentials could not be encoded.")
        }

        let updateStatus = SecItemUpdate(
            baseQuery as CFDictionary,
            [kSecValueData as String: data] as CFDictionary)
        if updateStatus == errSecSuccess {
            return
        }
        guard updateStatus == errSecItemNotFound else {
            throw sanitizedError(operation: "update")
        }

        var item = baseQuery
        item[kSecValueData as String] = data
        item[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        let addStatus = SecItemAdd(item as CFDictionary, nil)
        guard addStatus == errSecSuccess else {
            throw sanitizedError(operation: "save")
        }
    }

    func delete() throws {
        let status = SecItemDelete(baseQuery as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw sanitizedError(operation: "delete")
        }
    }

    private var baseQuery: [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: "primary",
        ]
    }

    private func sanitizedError(operation: String) -> AppError {
        AppError.localStorage("Secure cloud credential \(operation) failed.")
    }
}
