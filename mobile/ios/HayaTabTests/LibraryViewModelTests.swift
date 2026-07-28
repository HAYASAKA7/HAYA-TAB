import XCTest
@testable import HayaTab

@MainActor
final class LibraryViewModelTests: XCTestCase {
    func testLoadPublishesCachedItemsWhileRefreshIsPending() async {
        let cached = HayaTab.LibraryItem.viewModelFixture(
            title: "Cached",
            path: "scores/cached.pdf")
        let repository = ViewModelRepository(
            cached: [cached],
            refreshBehavior: .suspended)
        let viewModel = LibraryViewModel(repository: repository)
        let loadTask = Task { await viewModel.load() }

        await repository.waitUntilRefreshStarted()

        XCTAssertEqual(viewModel.items, [cached])
        XCTAssertEqual(viewModel.state, .loaded)

        loadTask.cancel()
        await loadTask.value
        XCTAssertEqual(viewModel.items, [cached])
        XCTAssertEqual(viewModel.state, .loaded)
    }

    func testRefreshFailureKeepsCachedItemsAndMarksLibraryOffline() async {
        let cached = HayaTab.LibraryItem.viewModelFixture(
            title: "Cached",
            path: "scores/cached.pdf")
        let error = AppError.transport("Cloud unavailable")
        let repository = ViewModelRepository(
            cached: [cached],
            refreshBehavior: .failure(error))
        let viewModel = LibraryViewModel(repository: repository)

        await viewModel.load()

        XCTAssertEqual(viewModel.items, [cached])
        XCTAssertEqual(viewModel.state, .offline(error))
    }

    func testRefreshFailureWithoutCacheShowsFailedState() async {
        let error = AppError.authentication
        let repository = ViewModelRepository(
            cached: [],
            refreshBehavior: .failure(error))
        let viewModel = LibraryViewModel(repository: repository)

        await viewModel.load()

        XCTAssertTrue(viewModel.items.isEmpty)
        XCTAssertEqual(viewModel.state, .failed(error))
    }

    func testManualRefreshReplacesVisibleItems() async {
        let fresh = HayaTab.LibraryItem.viewModelFixture(
            title: "Fresh",
            path: "scores/fresh.gp5",
            kind: .guitarPro)
        let repository = ViewModelRepository(
            cached: [],
            refreshBehavior: .success([fresh]))
        let viewModel = LibraryViewModel(repository: repository)

        await viewModel.refresh()

        XCTAssertEqual(viewModel.items, [fresh])
        XCTAssertEqual(viewModel.state, .loaded)
    }
}

private actor ViewModelRepository: LibraryRepositoryProtocol {
    enum RefreshBehavior: Sendable {
        case success([HayaTab.LibraryItem])
        case failure(AppError)
        case suspended
    }

    private let cached: [HayaTab.LibraryItem]
    private let refreshBehavior: RefreshBehavior
    private var refreshStarted = false
    private var refreshWaiters: [CheckedContinuation<Void, Never>] = []

    init(cached: [HayaTab.LibraryItem], refreshBehavior: RefreshBehavior) {
        self.cached = cached
        self.refreshBehavior = refreshBehavior
    }

    func cachedLibrary() async throws -> [HayaTab.LibraryItem] {
        cached
    }

    func refresh() async throws -> [HayaTab.LibraryItem] {
        refreshStarted = true
        refreshWaiters.forEach { $0.resume() }
        refreshWaiters.removeAll()

        switch refreshBehavior {
        case let .success(items):
            return items
        case let .failure(error):
            throw error
        case .suspended:
            try await Task.sleep(for: .seconds(30))
            return []
        }
    }

    func waitUntilRefreshStarted() async {
        if refreshStarted {
            return
        }
        await withCheckedContinuation { continuation in
            refreshWaiters.append(continuation)
        }
    }
}

private extension HayaTab.LibraryItem {
    static func viewModelFixture(
        title: String,
        path: String,
        kind: DocumentKind = .pdf
    ) -> HayaTab.LibraryItem {
        HayaTab.LibraryItem(
            id: HayaTab.LibraryItem.stableID(volumeID: "fixture-volume", relativePath: path),
            volumeID: "fixture-volume",
            relativePath: path,
            title: title,
            artist: "HAYA",
            album: "Tests",
            kind: kind,
            categories: ["Tests"],
            localFilename: nil,
            remoteRevision: "fixture")
    }
}
