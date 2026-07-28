import Foundation

struct AppEnvironment: Sendable {
    let libraryRepository: any LibraryRepositoryProtocol

    static func live() -> AppEnvironment {
        AppEnvironment(libraryRepository: EmptyLibraryRepository())
    }

    static func fixture() -> AppEnvironment {
        AppEnvironment(libraryRepository: FixtureLibraryRepository())
    }
}

private struct EmptyLibraryRepository: LibraryRepositoryProtocol {
    func cachedLibrary() async throws -> [LibraryItem] {
        []
    }

    func refresh() async throws -> [LibraryItem] {
        []
    }
}

private struct FixtureLibraryRepository: LibraryRepositoryProtocol {
    func cachedLibrary() async throws -> [LibraryItem] {
        [
            LibraryItem(
                id: LibraryItem.stableID(
                    volumeID: "volume-fixture",
                    relativePath: "scores/etude.gp5"),
                volumeID: "volume-fixture",
                relativePath: "scores/etude.gp5",
                title: "Etude",
                artist: "HAYA",
                album: "Practice",
                kind: .guitarPro,
                categories: ["Practice"],
                localFilename: nil,
                remoteRevision: nil),
            LibraryItem(
                id: LibraryItem.stableID(
                    volumeID: "volume-fixture",
                    relativePath: "scores/primer.pdf"),
                volumeID: "volume-fixture",
                relativePath: "scores/primer.pdf",
                title: "Primer",
                artist: "HAYA",
                album: "Reference",
                kind: .pdf,
                categories: ["Practice"],
                localFilename: nil,
                remoteRevision: nil),
        ]
    }

    func refresh() async throws -> [LibraryItem] {
        try await cachedLibrary()
    }
}
