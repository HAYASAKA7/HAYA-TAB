import Foundation
import Observation

enum LoadState: Equatable {
    case idle
    case loading
    case loaded
    case offline(AppError)
    case failed(AppError)
}

@MainActor
@Observable
final class LibraryViewModel {
    private let repository: any LibraryRepositoryProtocol
    private(set) var items: [LibraryItem] = []
    private(set) var state: LoadState = .idle

    init(repository: any LibraryRepositoryProtocol) {
        self.repository = repository
    }

    func load() async {
        guard state != .loading else { return }
        state = .loading

        do {
            items = try await repository.cachedLibrary()
            if !items.isEmpty {
                state = .loaded
            }
        } catch let error as AppError {
            state = .failed(error)
            return
        } catch {
            state = .failed(.localStorage("The cached library could not be read."))
            return
        }

        await refresh()
    }

    func refresh() async {
        do {
            let refreshedItems = try await repository.refresh()
            try Task.checkCancellation()
            items = refreshedItems
            state = .loaded
        } catch is CancellationError {
            state = items.isEmpty ? .idle : .loaded
        } catch let error as AppError {
            showRefreshFailure(error)
        } catch let error as WebDAVError {
            showRefreshFailure(mapWebDAVError(error))
        } catch {
            showRefreshFailure(
                .transport("The cloud library could not be refreshed."))
        }
    }

    private func showRefreshFailure(_ error: AppError) {
        state = items.isEmpty ? .failed(error) : .offline(error)
    }

    private func mapWebDAVError(_ error: WebDAVError) -> AppError {
        switch error {
        case .authenticationRequired:
            .authentication
        case .insecureTransport:
            .transport("The cloud account must use HTTPS.")
        case .remoteNotFound, .invalidRemotePath, .invalidResponse, .httpStatus, .transport:
            .transport("The cloud library could not be refreshed.")
        }
    }
}
