import Foundation
import Observation

enum DocumentDownloadState: Equatable {
    case queued
    case downloading
    case availableOffline
    case failed(AppError)
}

struct ReaderSelection: Identifiable, Hashable {
    let item: LibraryItem
    let documentURL: URL

    var id: String { item.id }
}

@MainActor
@Observable
final class DownloadViewModel {
    private let store: any DownloadStoring
    private(set) var states: [String: DocumentDownloadState] = [:]
    private(set) var itemsByID: [String: LibraryItem] = [:]
    var readerSelection: ReaderSelection?
    @ObservationIgnored private var tasks: [String: Task<Void, Never>] = [:]

    init(store: any DownloadStoring) {
        self.store = store
    }

    var displayItems: [LibraryItem] {
        itemsByID.values
            .filter { states[$0.id] != nil }
            .sorted {
                $0.title.localizedStandardCompare($1.title) == .orderedAscending
            }
    }

    func state(for item: LibraryItem) -> DocumentDownloadState? {
        states[item.id] ?? (item.localFilename == nil ? nil : .availableOffline)
    }

    func restore() async {
        do {
            let items = try await store.offlineItems()
            for item in items {
                itemsByID[item.id] = item
                states[item.id] = .availableOffline
            }
        } catch {
            // A failed restore is represented by the library's typed storage state.
        }
    }

    func start(_ item: LibraryItem) {
        tasks[item.id]?.cancel()
        itemsByID[item.id] = item
        states[item.id] = .queued

        tasks[item.id] = Task { [weak self] in
            guard let self else { return }
            await Task.yield()
            guard !Task.isCancelled else {
                self.finishCancellation(for: item)
                return
            }
            self.states[item.id] = .downloading

            do {
                _ = try await self.store.download(item)
                try Task.checkCancellation()
                self.states[item.id] = .availableOffline
                await self.restore()
            } catch is CancellationError {
                self.finishCancellation(for: item)
            } catch let error as AppError {
                self.states[item.id] = .failed(error)
            } catch {
                self.states[item.id] = .failed(
                    .transport("The document could not be downloaded."))
            }
            self.tasks[item.id] = nil
        }
    }

    func cancel(_ item: LibraryItem) {
        tasks[item.id]?.cancel()
    }

    func delete(_ item: LibraryItem) async {
        tasks[item.id]?.cancel()
        do {
            try await store.delete(item)
            states[item.id] = nil
            itemsByID[item.id] = nil
        } catch let error as AppError {
            states[item.id] = .failed(error)
        } catch {
            states[item.id] = .failed(
                .localStorage("The offline document could not be deleted."))
        }
    }

    func open(_ item: LibraryItem) async {
        do {
            guard let documentURL = try await store.localURL(for: item) else {
                states[item.id] = .failed(
                    .localStorage("The offline document is unavailable."))
                return
            }
            readerSelection = ReaderSelection(
                item: item,
                documentURL: documentURL)
        } catch let error as AppError {
            states[item.id] = .failed(error)
        } catch {
            states[item.id] = .failed(
                .localStorage("The offline document could not be opened."))
        }
    }

    private func finishCancellation(for item: LibraryItem) {
        if item.localFilename == nil {
            states[item.id] = nil
            itemsByID[item.id] = nil
        } else {
            states[item.id] = .availableOffline
        }
    }
}
