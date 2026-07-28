import Foundation
import Observation

enum LoadState: Equatable {
    case idle
    case loading
    case loaded
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
            state = .loaded
        } catch let error as AppError {
            state = .failed(error)
        } catch {
            state = .failed(.localStorage(error.localizedDescription))
        }
    }
}
