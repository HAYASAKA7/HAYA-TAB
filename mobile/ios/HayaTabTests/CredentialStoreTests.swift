import Foundation
import XCTest
@testable import HayaTab

final class CredentialStoreTests: XCTestCase {
    func testSaveLoadAndDeleteCredential() throws {
        let store = CredentialStore(
            service: "com.hayasaka7.hayatab.tests.\(UUID().uuidString)")
        defer { try? store.delete() }

        let credential = WebDAVCredential(
            baseURL: try XCTUnwrap(URL(string: "https://cloud.example.test/music/")),
            username: "haya",
            password: "test-only-secret")

        try store.save(credential)
        let loaded = try XCTUnwrap(store.load())

        XCTAssertEqual(loaded.baseURL, credential.baseURL)
        XCTAssertEqual(loaded.username, credential.username)
        XCTAssertEqual(loaded.password.count, credential.password.count)

        try store.delete()
        XCTAssertNil(try store.load())
    }

    func testMissingCredentialReturnsNil() throws {
        let store = CredentialStore(
            service: "com.hayasaka7.hayatab.tests.\(UUID().uuidString)")
        XCTAssertNil(try store.load())
    }
}
