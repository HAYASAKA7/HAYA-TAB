import Foundation
import SwiftData
import XCTest
@testable import HayaTab

final class DownloadStoreTests: XCTestCase {
    func testSuccessfulDownloadPromotesStableFileAndPersistsLocalFilename() async throws {
        let harness = try await makeHarness(
            behavior: .success(
                data: Data("document".utf8),
                expectedByteCount: 8,
                validatorETag: "\"v1\"",
                responseETag: "\"v1\""))

        let localURL = try await harness.downloadStore.download(harness.item)

        XCTAssertEqual(try Data(contentsOf: localURL), Data("document".utf8))
        XCTAssertEqual(localURL.lastPathComponent, "\(harness.item.id).pdf")
        let restored = try await harness.libraryStore.all()
        XCTAssertEqual(restored.first?.localFilename, localURL.lastPathComponent)
    }

    func testCancellationRemovesOnlyPartialFile() async throws {
        let harness = try await makeHarness(
            behavior: .cancellation(afterWriting: Data("partial".utf8)))

        do {
            _ = try await harness.downloadStore.download(harness.item)
            XCTFail("Expected cancellation")
        } catch is CancellationError {
            // Cancellation must remain distinguishable to the UI.
        }

        XCTAssertTrue(try partialFiles(in: harness.rootURL).isEmpty)
        XCTAssertFalse(FileManager.default.fileExists(
            atPath: harness.rootURL.appendingPathComponent("\(harness.item.id).pdf").path))
        let restored = try await harness.libraryStore.all()
        XCTAssertNil(restored.first?.localFilename)
    }

    func testExpectedByteCountMismatchRejectsPromotion() async throws {
        let harness = try await makeHarness(
            behavior: .success(
                data: Data("short".utf8),
                expectedByteCount: 100,
                validatorETag: nil,
                responseETag: nil))

        await XCTAssertThrowsAppError(.downloadIntegrity) {
            _ = try await harness.downloadStore.download(harness.item)
        }

        XCTAssertTrue(try partialFiles(in: harness.rootURL).isEmpty)
        XCTAssertFalse(FileManager.default.fileExists(
            atPath: harness.rootURL.appendingPathComponent("\(harness.item.id).pdf").path))
    }

    func testStalePartialFilesAreRemovedBeforeDownload() async throws {
        let harness = try await makeHarness(
            behavior: .success(
                data: Data("fresh".utf8),
                expectedByteCount: 5,
                validatorETag: nil,
                responseETag: nil))
        let staleURL = harness.rootURL.appendingPathComponent(
            "\(harness.item.id).stale.partial")
        try Data("stale".utf8).write(to: staleURL)

        _ = try await harness.downloadStore.download(harness.item)

        XCTAssertFalse(FileManager.default.fileExists(atPath: staleURL.path))
        XCTAssertTrue(try partialFiles(in: harness.rootURL).isEmpty)
    }

    func testFailedReplacementPreservesExistingValidFileAndRecord() async throws {
        let harness = try await makeHarness(
            behavior: .failure(
                afterWriting: Data("new-partial".utf8),
                error: .transport("Cloud unavailable")),
            seedLocalFile: Data("old-valid".utf8))
        let initialRecords = try await harness.libraryStore.all()
        let filename = try XCTUnwrap(
            initialRecords.first?.localFilename)
        let existingURL = harness.rootURL.appendingPathComponent(filename)

        await XCTAssertThrowsAppError(.transport("Cloud unavailable")) {
            _ = try await harness.downloadStore.download(harness.item)
        }

        XCTAssertEqual(try Data(contentsOf: existingURL), Data("old-valid".utf8))
        let restoredRecords = try await harness.libraryStore.all()
        XCTAssertEqual(
            restoredRecords.first?.localFilename,
            filename)
        XCTAssertTrue(try partialFiles(in: harness.rootURL).isEmpty)
    }

    func testResponseETagMismatchRejectsNewBytesAndPreservesExistingFile() async throws {
        let harness = try await makeHarness(
            behavior: .success(
                data: Data("changed".utf8),
                expectedByteCount: 7,
                validatorETag: "\"v1\"",
                responseETag: "\"v2\""),
            seedLocalFile: Data("old-valid".utf8))
        let initialRecords = try await harness.libraryStore.all()
        let filename = try XCTUnwrap(
            initialRecords.first?.localFilename)
        let existingURL = harness.rootURL.appendingPathComponent(filename)

        await XCTAssertThrowsAppError(.remoteChanged) {
            _ = try await harness.downloadStore.download(harness.item)
        }

        XCTAssertEqual(try Data(contentsOf: existingURL), Data("old-valid".utf8))
        XCTAssertTrue(try partialFiles(in: harness.rootURL).isEmpty)
    }

    func testOfflineItemsRestoreOnlyPersistedFilesThatStillExist() async throws {
        let harness = try await makeHarness(
            behavior: .success(
                data: Data("unused".utf8),
                expectedByteCount: 6,
                validatorETag: nil,
                responseETag: nil),
            seedLocalFile: Data("offline".utf8))

        let restored = try await harness.downloadStore.offlineItems()

        XCTAssertEqual(restored.count, 1)
        XCTAssertEqual(restored.first?.id, harness.item.id)
        XCTAssertNotNil(restored.first?.localFilename)
    }

    private func makeHarness(
        behavior: DownloadStub.Behavior,
        seedLocalFile: Data? = nil
    ) async throws -> Harness {
        let rootURL = FileManager.default.temporaryDirectory
            .appendingPathComponent("haya-download-tests-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(
            at: rootURL,
            withIntermediateDirectories: true)
        addTeardownBlock {
            try? FileManager.default.removeItem(at: rootURL)
        }

        let configuration = ModelConfiguration(isStoredInMemoryOnly: true)
        let container = try ModelContainer(
            for: LibraryRecord.self,
            configurations: configuration)
        let libraryStore = LibraryStore(container: container)
        let baseItem = HayaTab.LibraryItem.downloadFixture()
        let item: HayaTab.LibraryItem
        if let seedLocalFile {
            let filename = "\(baseItem.id).pdf"
            try seedLocalFile.write(to: rootURL.appendingPathComponent(filename))
            item = HayaTab.LibraryItem(
                id: baseItem.id,
                volumeID: baseItem.volumeID,
                relativePath: baseItem.relativePath,
                title: baseItem.title,
                artist: baseItem.artist,
                album: baseItem.album,
                kind: baseItem.kind,
                categories: baseItem.categories,
                localFilename: filename,
                remoteRevision: baseItem.remoteRevision)
        } else {
            item = baseItem
        }
        try await libraryStore.replace(with: [item])

        return Harness(
            rootURL: rootURL,
            item: item,
            libraryStore: libraryStore,
            downloadStore: DownloadStore(
                rootURL: rootURL,
                downloader: DownloadStub(behavior: behavior),
                libraryStore: libraryStore))
    }

    private func partialFiles(in rootURL: URL) throws -> [URL] {
        try FileManager.default.contentsOfDirectory(
            at: rootURL,
            includingPropertiesForKeys: nil)
            .filter { $0.pathExtension == "partial" }
    }
}

private struct Harness {
    let rootURL: URL
    let item: HayaTab.LibraryItem
    let libraryStore: LibraryStore
    let downloadStore: DownloadStore
}

private actor DownloadStub: DocumentDownloading {
    enum Behavior: Sendable {
        case success(
            data: Data,
            expectedByteCount: Int64?,
            validatorETag: String?,
            responseETag: String?)
        case cancellation(afterWriting: Data)
        case failure(afterWriting: Data, error: AppError)
    }

    let behavior: Behavior

    init(behavior: Behavior) {
        self.behavior = behavior
    }

    func download(path: String, to partialURL: URL) async throws -> DownloadTransfer {
        switch behavior {
        case let .success(data, expectedByteCount, validatorETag, responseETag):
            try data.write(to: partialURL)
            return DownloadTransfer(
                expectedByteCount: expectedByteCount,
                validatorETag: validatorETag,
                responseETag: responseETag)
        case let .cancellation(data):
            try data.write(to: partialURL)
            throw CancellationError()
        case let .failure(data, error):
            try data.write(to: partialURL)
            throw error
        }
    }
}

private extension HayaTab.LibraryItem {
    static func downloadFixture() -> HayaTab.LibraryItem {
        let path = "scores/primer.pdf"
        return HayaTab.LibraryItem(
            id: HayaTab.LibraryItem.stableID(
                volumeID: "fixture-volume",
                relativePath: path),
            volumeID: "fixture-volume",
            relativePath: path,
            title: "Primer",
            artist: "HAYA",
            album: "Tests",
            kind: .pdf,
            categories: ["Tests"],
            localFilename: nil,
            remoteRevision: "2026-07-28T00:00:00Z")
    }
}

private func XCTAssertThrowsAppError(
    _ expected: AppError,
    operation: () async throws -> Void,
    file: StaticString = #filePath,
    line: UInt = #line
) async {
    do {
        try await operation()
        XCTFail("Expected \(expected)", file: file, line: line)
    } catch let error as AppError {
        XCTAssertEqual(error, expected, file: file, line: line)
    } catch {
        XCTFail("Expected AppError, got \(type(of: error))", file: file, line: line)
    }
}
