import Foundation
import XCTest
@testable import HayaTab

@MainActor
final class RestorationTests: XCTestCase {
    func testUnknownRootDestinationFallsBackToLibrary() {
        XCTAssertEqual(
            RootDestination.restored(from: "Settings"),
            .settings)
        XCTAssertEqual(
            RootDestination.restored(from: "removed-destination"),
            .library)
        XCTAssertEqual(
            RootDestination.restored(from: nil),
            .library)
    }

    func testReaderRestoresOnlyWhenValidatedOfflineFileExists() async {
        let item = LibraryItem.restorationFixture()
        let documentURL = URL(fileURLWithPath: "/validated/offline/primer.pdf")
        let store = RestorationDownloadStore(
            items: [item],
            localURLs: [item.id: documentURL])
        let viewModel = DownloadViewModel(store: store)

        await viewModel.restoreReader(itemID: item.id)

        XCTAssertEqual(
            viewModel.readerSelection,
            ReaderSelection(item: item, documentURL: documentURL))
    }

    func testReaderDoesNotRestoreWhenOfflineFileIsMissing() async {
        let item = LibraryItem.restorationFixture()
        let store = RestorationDownloadStore(
            items: [item],
            localURLs: [:])
        let viewModel = DownloadViewModel(store: store)

        await viewModel.restoreReader(itemID: item.id)

        XCTAssertNil(viewModel.readerSelection)
        XCTAssertNil(viewModel.state(for: item))
    }

    func testReaderDoesNotRestoreUnknownLibraryItem() async {
        let item = LibraryItem.restorationFixture()
        let store = RestorationDownloadStore(
            items: [item],
            localURLs: [item.id: URL(fileURLWithPath: "/validated/offline/primer.pdf")])
        let viewModel = DownloadViewModel(store: store)

        await viewModel.restoreReader(itemID: "unknown-item")

        XCTAssertNil(viewModel.readerSelection)
    }
}

private actor RestorationDownloadStore: DownloadStoring {
    let items: [LibraryItem]
    let localURLs: [String: URL]

    init(items: [LibraryItem], localURLs: [String: URL]) {
        self.items = items
        self.localURLs = localURLs
    }

    func download(_ item: LibraryItem) async throws -> URL {
        throw AppError.transport("Not used by restoration tests.")
    }

    func offlineItems() async throws -> [LibraryItem] {
        items
    }

    func delete(_ item: LibraryItem) async throws {}

    func localURL(for item: LibraryItem) async throws -> URL? {
        localURLs[item.id]
    }
}

private extension LibraryItem {
    static func restorationFixture() -> LibraryItem {
        let path = "scores/primer.pdf"
        let id = stableID(
            volumeID: "fixture-volume",
            relativePath: path)
        return LibraryItem(
            id: id,
            volumeID: "fixture-volume",
            relativePath: path,
            title: "Primer",
            artist: "HAYA",
            album: "Tests",
            kind: .pdf,
            categories: ["Tests"],
            localFilename: "\(id).pdf",
            remoteRevision: "2026-07-28T00:00:00Z")
    }
}
