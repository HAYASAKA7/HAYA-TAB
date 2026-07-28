import Foundation
import SwiftData

struct AppEnvironment: Sendable {
    let libraryRepository: any LibraryRepositoryProtocol
    let credentialStore: any CredentialStoring
    let clientFactory: any WebDAVClientBuilding
    let downloadStore: any DownloadStoring

    static func live() -> AppEnvironment {
        let credentialStore = CredentialStore()
        let clientFactory = WebDAVClientFactory()

        do {
            let container = try ModelContainer(for: LibraryRecord.self)
            let store = LibraryStore(container: container)
            let webDAV = CredentialBackedWebDAV(
                credentialStore: credentialStore,
                clientFactory: clientFactory)
            let downloader = CredentialBackedDocumentDownloader(
                credentialStore: credentialStore)
            let supportURL = FileManager.default.urls(
                for: .applicationSupportDirectory,
                in: .userDomainMask).first
            guard let supportURL else {
                throw AppError.localStorage(
                    "The application support directory is unavailable.")
            }
            return AppEnvironment(
                libraryRepository: LibraryRepository(store: store, webDAV: webDAV),
                credentialStore: credentialStore,
                clientFactory: clientFactory,
                downloadStore: DownloadStore(
                    rootURL: supportURL.appendingPathComponent(
                        "Documents",
                        isDirectory: true),
                    downloader: downloader,
                    libraryStore: store))
        } catch {
            return unavailable(
                credentialStore: credentialStore,
                clientFactory: clientFactory)
        }
    }

    static func fixture() -> AppEnvironment {
        AppEnvironment(
            libraryRepository: FixtureLibraryRepository(),
            credentialStore: FixtureCredentialStore(),
            clientFactory: FixtureWebDAVClientFactory(),
            downloadStore: UnavailableDownloadStore(
                error: .transport("Fixture documents are not downloadable.")))
    }
}

private struct UnavailableLibraryRepository: LibraryRepositoryProtocol {
    func cachedLibrary() async throws -> [LibraryItem] {
        throw AppError.localStorage("The local library could not be opened.")
    }

    func refresh() async throws -> [LibraryItem] {
        throw AppError.localStorage("The local library could not be opened.")
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

private struct FixtureCredentialStore: CredentialStoring {
    func load() throws -> WebDAVCredential? {
        nil
    }

    func save(_ credential: WebDAVCredential) throws {}

    func delete() throws {}
}

private struct FixtureWebDAVClientFactory: WebDAVClientBuilding {
    func makeClient(credential: WebDAVCredential) throws -> any WebDAVServing {
        FixtureWebDAVServer()
    }
}

private struct FixtureWebDAVServer: WebDAVServing {
    func get(path: String) async throws -> WebDAVResponse {
        throw WebDAVError.remoteNotFound
    }

    func testConnection() async throws {}
}

private actor CredentialBackedWebDAV: WebDAVServing {
    private let credentialStore: any CredentialStoring
    private let clientFactory: any WebDAVClientBuilding

    init(
        credentialStore: any CredentialStoring,
        clientFactory: any WebDAVClientBuilding
    ) {
        self.credentialStore = credentialStore
        self.clientFactory = clientFactory
    }

    func get(path: String) async throws -> WebDAVResponse {
        let client = try configuredClient()
        return try await client.get(path: path)
    }

    func testConnection() async throws {
        let client = try configuredClient()
        try await client.testConnection()
    }

    private func configuredClient() throws -> any WebDAVServing {
        guard let credential = try credentialStore.load() else {
            throw AppError.authentication
        }
        return try clientFactory.makeClient(credential: credential)
    }
}

private actor CredentialBackedDocumentDownloader: DocumentDownloading {
    private let credentialStore: any CredentialStoring

    init(credentialStore: any CredentialStoring) {
        self.credentialStore = credentialStore
    }

    func download(path: String, to partialURL: URL) async throws -> DownloadTransfer {
        guard let credential = try credentialStore.load() else {
            throw AppError.authentication
        }
        let client = try WebDAVClient(credential: credential)
        return try await client.download(path: path, to: partialURL)
    }
}

private struct UnavailableDownloadStore: DownloadStoring {
    let error: AppError

    func download(_ item: LibraryItem) async throws -> URL {
        throw error
    }

    func offlineItems() async throws -> [LibraryItem] {
        []
    }

    func delete(_ item: LibraryItem) async throws {
        throw error
    }

    func localURL(for item: LibraryItem) async throws -> URL? {
        nil
    }
}

private extension AppEnvironment {
    static func unavailable(
        credentialStore: any CredentialStoring,
        clientFactory: any WebDAVClientBuilding
    ) -> AppEnvironment {
        let error = AppError.localStorage("The local library could not be opened.")
        return AppEnvironment(
            libraryRepository: UnavailableLibraryRepository(),
            credentialStore: credentialStore,
            clientFactory: clientFactory,
            downloadStore: UnavailableDownloadStore(error: error))
    }
}
