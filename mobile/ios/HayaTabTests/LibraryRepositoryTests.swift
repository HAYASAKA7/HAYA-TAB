import Foundation
import SwiftData
import XCTest
@testable import HayaTab

final class LibraryRepositoryTests: XCTestCase {
    func testCachedLibraryReturnsStoredRecordsWithoutNetwork() async throws {
        let cached = HayaTab.LibraryItem.cachedFixture()
        let store = try await makeStore(seed: [cached])
        let webDAV = WebDAVStub()
        let repository = LibraryRepository(store: store, webDAV: webDAV)

        let items = try await repository.cachedLibrary()
        let requestedPaths = await webDAV.requestedPathsSnapshot()

        XCTAssertEqual(items, [cached])
        XCTAssertTrue(requestedPaths.isEmpty)
    }

    func testRefreshFetchesAllSixteenBucketsAndMergesDocuments() async throws {
        let store = try await makeStore()
        let responses = try manifestResponses(filesByBucket: [
            0: [fingerprintFile(path: "scores/primer.pdf", title: "Primer", type: "pdf")],
            3: [fingerprintFile(path: "scores/etude.gp5", title: "Etude", type: "gp")],
        ])
        let webDAV = WebDAVStub(responses: responses)
        let repository = LibraryRepository(store: store, webDAV: webDAV)

        let items = try await repository.refresh()
        let requestedPaths = Set(await webDAV.requestedPathsSnapshot())
        let expectedPaths = Set((0 ... 15).map(Self.bucketPath))

        XCTAssertEqual(Set(items.map(\.relativePath)), ["scores/primer.pdf", "scores/etude.gp5"])
        XCTAssertEqual(requestedPaths, expectedPaths)
        let persisted = try await store.all()
        XCTAssertEqual(persisted, items)
    }

    func testMissingNonzeroBucketsAreTreatedAsEmpty() async throws {
        let store = try await makeStore()
        let bucketZero = try response(
            bucket: 0,
            files: [fingerprintFile(path: "scores/primer.pdf", title: "Primer", type: "pdf")])
        let webDAV = WebDAVStub(responses: [Self.bucketPath(0): bucketZero])
        let repository = LibraryRepository(store: store, webDAV: webDAV)

        let items = try await repository.refresh()

        XCTAssertEqual(items.map(\.relativePath), ["scores/primer.pdf"])
    }

    func testRefreshLimitsConcurrentBucketRequestsToFour() async throws {
        let store = try await makeStore()
        let webDAV = WebDAVStub(
            responses: try manifestResponses(),
            requestDelay: .milliseconds(20))
        let repository = LibraryRepository(store: store, webDAV: webDAV)

        _ = try await repository.refresh()
        let maximumConcurrentRequests = await webDAV.maximumConcurrentRequestCount()

        XCTAssertEqual(maximumConcurrentRequests, 4)
    }

    func testMalformedBucketLeavesCachedDatabaseUntouched() async throws {
        let cached = HayaTab.LibraryItem.cachedFixture()
        let store = try await makeStore(seed: [cached])
        var responses = try manifestResponses()
        responses[Self.bucketPath(3)] = WebDAVResponse(
            statusCode: 200,
            data: Data("{malformed".utf8),
            etag: "malformed")
        let repository = LibraryRepository(store: store, webDAV: WebDAVStub(responses: responses))

        await XCTAssertThrowsAppError(.malformedManifest) {
            _ = try await repository.refresh()
        }
        let restored = try await store.all()

        XCTAssertEqual(restored, [cached])
    }

    func testDuplicatePathUsesNewestRevisionDeterministically() async throws {
        let store = try await makeStore()
        let older = fingerprintFile(
            path: "scores/primer.pdf",
            title: "Older",
            type: "pdf",
            uploadedAt: "2026-07-27T00:00:00Z")
        let newer = fingerprintFile(
            path: "scores/primer.pdf",
            title: "Newer",
            type: "pdf",
            uploadedAt: "2026-07-28T00:00:00Z")
        let responses = try manifestResponses(filesByBucket: [0: [older], 15: [newer]])
        let repository = LibraryRepository(
            store: store,
            webDAV: WebDAVStub(responses: responses))

        let items = try await repository.refresh()

        XCTAssertEqual(items.count, 1)
        XCTAssertEqual(items.first?.title, "Newer")
        XCTAssertEqual(items.first?.remoteRevision, "2026-07-28T00:00:00Z")
    }

    func testCancellationLeavesCachedDatabaseUntouched() async throws {
        let cached = HayaTab.LibraryItem.cachedFixture()
        let store = try await makeStore(seed: [cached])
        let responses = try manifestResponses()
        let blockedPath = Self.bucketPath(1)
        let webDAV = WebDAVStub(responses: responses, blockedPaths: [blockedPath])
        let repository = LibraryRepository(store: store, webDAV: webDAV)
        let refreshTask = Task { try await repository.refresh() }

        await webDAV.waitUntilRequested(blockedPath)
        refreshTask.cancel()

        do {
            _ = try await refreshTask.value
            XCTFail("Expected refresh cancellation")
        } catch is CancellationError {
            // Cancellation is deliberately preserved for the UI task lifecycle.
        } catch {
            XCTFail("Expected CancellationError, got \(type(of: error))")
        }
        let restored = try await store.all()
        XCTAssertEqual(restored, [cached])
    }

    func testUnsafeRelativePathLeavesCachedDatabaseUntouched() async throws {
        let cached = HayaTab.LibraryItem.cachedFixture()
        let store = try await makeStore(seed: [cached])
        let bucketZero = try response(
            bucket: 0,
            files: [fingerprintFile(path: "scores/../escape.pdf", title: "Escape", type: "pdf")])
        let repository = LibraryRepository(
            store: store,
            webDAV: WebDAVStub(responses: [Self.bucketPath(0): bucketZero]))

        await XCTAssertThrowsAppError(.unsafeRemotePath("scores/../escape.pdf")) {
            _ = try await repository.refresh()
        }
        let restored = try await store.all()
        XCTAssertEqual(restored, [cached])
    }

    func testUnsupportedExtensionLeavesCachedDatabaseUntouched() async throws {
        let cached = HayaTab.LibraryItem.cachedFixture()
        let store = try await makeStore(seed: [cached])
        let bucketZero = try response(
            bucket: 0,
            files: [fingerprintFile(path: "scores/readme.txt", title: "Read Me", type: "pdf")])
        let repository = LibraryRepository(
            store: store,
            webDAV: WebDAVStub(responses: [Self.bucketPath(0): bucketZero]))

        await XCTAssertThrowsAppError(.unsupportedDocument("scores/readme.txt")) {
            _ = try await repository.refresh()
        }
        let restored = try await store.all()
        XCTAssertEqual(restored, [cached])
    }

    private func makeStore(seed: [HayaTab.LibraryItem] = []) async throws -> LibraryStore {
        let configuration = ModelConfiguration(isStoredInMemoryOnly: true)
        let container = try ModelContainer(
            for: LibraryRecord.self,
            configurations: configuration)
        let store = LibraryStore(container: container)
        if !seed.isEmpty {
            try await store.replace(with: seed)
        }
        return store
    }

    private func manifestResponses(
        filesByBucket: [Int: [[String: Any]]] = [:]
    ) throws -> [String: WebDAVResponse] {
        try Dictionary(uniqueKeysWithValues: (0 ... 15).map { number in
            (
                Self.bucketPath(number),
                try response(bucket: number, files: filesByBucket[number] ?? []))
        })
    }

    private func response(
        bucket: Int,
        files: [[String: Any]],
        volumeID: String = "volume-fixture"
    ) throws -> WebDAVResponse {
        let object: [String: Any]
        if bucket == 0 {
            object = [
                "metadata": [
                    "volume_id": volumeID,
                    "volume_name": "Fixture Volume",
                    "created_at": "2026-07-28T00:00:00Z",
                    "app_version": "3.1.7",
                    "device_name": "fixture",
                    "last_updated": "2026-07-28T00:00:00Z",
                    "bucket_count": 16,
                ],
                "files": files,
            ]
        } else {
            object = ["bucket_number": bucket, "files": files]
        }
        return WebDAVResponse(
            statusCode: 200,
            data: try JSONSerialization.data(withJSONObject: object, options: [.sortedKeys]),
            etag: "bucket-\(bucket)")
    }

    private func fingerprintFile(
        path: String,
        title: String,
        type: String,
        uploadedAt: String = "2026-07-28T00:00:00Z"
    ) -> [String: Any] {
        [
            "relative_path": path,
            "title": title,
            "artist": "HAYA",
            "album": "Practice",
            "type": type,
            "categories": ["Practice"],
            "uploaded_at": uploadedAt,
            "uploaded_by": "fixture",
        ]
    }

    private static func bucketPath(_ number: Int) -> String {
        String(format: "haya-metadata/bucket-%02d.json", number)
    }
}

private actor WebDAVStub: WebDAVServing {
    private let responses: [String: WebDAVResponse]
    private let failures: [String: WebDAVError]
    private let blockedPaths: Set<String>
    private let requestDelay: Duration
    private var requestedPaths: [String] = []
    private var requestWaiters: [String: [CheckedContinuation<Void, Never>]] = [:]
    private var activeRequestCount = 0
    private var maximumActiveRequestCount = 0

    init(
        responses: [String: WebDAVResponse] = [:],
        failures: [String: WebDAVError] = [:],
        blockedPaths: Set<String> = [],
        requestDelay: Duration = .zero
    ) {
        self.responses = responses
        self.failures = failures
        self.blockedPaths = blockedPaths
        self.requestDelay = requestDelay
    }

    func get(path: String) async throws -> WebDAVResponse {
        requestedPaths.append(path)
        requestWaiters.removeValue(forKey: path)?.forEach { $0.resume() }
        activeRequestCount += 1
        maximumActiveRequestCount = max(maximumActiveRequestCount, activeRequestCount)
        defer { activeRequestCount -= 1 }

        if blockedPaths.contains(path) {
            try await Task.sleep(for: .seconds(30))
        } else if requestDelay != .zero {
            try await Task.sleep(for: requestDelay)
        }
        if let failure = failures[path] {
            throw failure
        }
        guard let response = responses[path] else {
            throw WebDAVError.remoteNotFound
        }
        return response
    }

    func testConnection() async throws {}

    func requestedPathsSnapshot() -> [String] {
        requestedPaths
    }

    func maximumConcurrentRequestCount() -> Int {
        maximumActiveRequestCount
    }

    func waitUntilRequested(_ path: String) async {
        if requestedPaths.contains(path) {
            return
        }
        await withCheckedContinuation { continuation in
            requestWaiters[path, default: []].append(continuation)
        }
    }
}

private extension HayaTab.LibraryItem {
    static func cachedFixture() -> HayaTab.LibraryItem {
        HayaTab.LibraryItem(
            id: HayaTab.LibraryItem.stableID(
                volumeID: "cached-volume",
                relativePath: "cached/score.pdf"),
            volumeID: "cached-volume",
            relativePath: "cached/score.pdf",
            title: "Cached score",
            artist: "HAYA",
            album: "Offline",
            kind: .pdf,
            categories: ["Cached"],
            localFilename: nil,
            remoteRevision: "cached-revision")
    }
}

private func XCTAssertThrowsAppError(
    _ expected: AppError,
    _ expression: () async throws -> Void,
    file: StaticString = #filePath,
    line: UInt = #line
) async {
    do {
        try await expression()
        XCTFail("Expected \(expected)", file: file, line: line)
    } catch let error as AppError {
        XCTAssertEqual(error, expected, file: file, line: line)
    } catch {
        XCTFail(
            "Expected AppError, got \(type(of: error))",
            file: file,
            line: line)
    }
}
