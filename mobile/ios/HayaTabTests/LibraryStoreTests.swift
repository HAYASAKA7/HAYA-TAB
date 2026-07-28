import SwiftData
import XCTest
@testable import HayaTab

final class LibraryStoreTests: XCTestCase {
    func testReplaceThenRestoreLibrary() async throws {
        let configuration = ModelConfiguration(isStoredInMemoryOnly: true)
        let container = try ModelContainer(
            for: LibraryRecord.self,
            configurations: configuration)
        let store = LibraryStore(container: container)
        let expected = LibraryItem.fixture()

        try await store.replace(with: [expected])

        let restored = try await store.all()
        XCTAssertEqual(restored, [expected])
    }

    func testReplaceRemovesRecordsMissingFromCompleteManifest() async throws {
        let configuration = ModelConfiguration(isStoredInMemoryOnly: true)
        let container = try ModelContainer(
            for: LibraryRecord.self,
            configurations: configuration)
        let store = LibraryStore(container: container)
        try await store.replace(with: [.fixture(), .secondFixture()])

        try await store.replace(with: [.fixture()])

        XCTAssertEqual(try await store.all(), [.fixture()])
    }

    func testStableIdentifierMatchesCrossPlatformSHA256Contract() {
        XCTAssertEqual(
            LibraryItem.stableID(
                volumeID: "volume-fixture",
                relativePath: "scores/etude.gp5"),
            "80e98ff31b161a58f7712ef844bff919978698397709ffdfb5c9b5aa4be4eec9")
    }
}

private extension LibraryItem {
    static func fixture() -> LibraryItem {
        LibraryItem(
            id: "volume-fixture:etude",
            volumeID: "volume-fixture",
            relativePath: "scores/etude.gp5",
            title: "Etude",
            artist: "HAYA",
            album: "",
            kind: .guitarPro,
            categories: ["Practice"],
            localFilename: nil,
            remoteRevision: nil)
    }

    static func secondFixture() -> LibraryItem {
        LibraryItem(
            id: "volume-fixture:primer",
            volumeID: "volume-fixture",
            relativePath: "scores/primer.pdf",
            title: "Primer",
            artist: "HAYA",
            album: "",
            kind: .pdf,
            categories: [],
            localFilename: nil,
            remoteRevision: nil)
    }
}
