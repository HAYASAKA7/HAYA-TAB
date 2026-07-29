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
        let etudePath = "scores/etude.gp5"
        let etude = LibraryItem(
            id: LibraryItem.stableID(
                volumeID: "volume-fixture",
                relativePath: etudePath),
            volumeID: "volume-fixture",
            relativePath: etudePath,
            title: "Etude",
            artist: "HAYA",
            album: "Practice",
            kind: .guitarPro,
            categories: ["Practice"],
            localFilename: nil,
            remoteRevision: nil)

        let primerPath = "scores/primer.pdf"
        let primerID = LibraryItem.stableID(
            volumeID: "volume-fixture",
            relativePath: primerPath)
        let primer = LibraryItem(
            id: primerID,
            volumeID: "volume-fixture",
            relativePath: primerPath,
            title: "Primer",
            artist: "HAYA",
            album: "Reference",
            kind: .pdf,
            categories: ["Practice"],
            localFilename: "\(primerID).pdf",
            remoteRevision: nil)

        let fixtureRoot = FileManager.default.temporaryDirectory
            .appendingPathComponent("haya-tab-ui-fixture", isDirectory: true)
        let primerURL = fixtureRoot
            .appendingPathComponent("\(primerID).pdf", isDirectory: false)
        try? FileManager.default.createDirectory(
            at: fixtureRoot,
            withIntermediateDirectories: true)
        try? Data("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n".utf8)
            .write(to: primerURL, options: .atomic)

        return AppEnvironment(
            libraryRepository: FixtureLibraryRepository(items: [etude, primer]),
            credentialStore: FixtureCredentialStore(),
            clientFactory: FixtureWebDAVClientFactory(),
            downloadStore: FixtureDownloadStore(
                offlineItem: primer,
                documentURL: primerURL))
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
    let items: [LibraryItem]

    func cachedLibrary() async throws -> [LibraryItem] {
        items
    }

    func refresh() async throws -> [LibraryItem] {
        items
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

private actor FixtureDownloadStore: DownloadStoring {
    let offlineItem: LibraryItem
    let documentURL: URL

    init(offlineItem: LibraryItem, documentURL: URL) {
        self.offlineItem = offlineItem
        self.documentURL = documentURL
    }

    func download(_ item: LibraryItem) async throws -> URL {
        throw AppError.transport("Fixture documents are not downloadable.")
    }

    func offlineItems() async throws -> [LibraryItem] {
        isValidDocument ? [offlineItem] : []
    }

    func delete(_ item: LibraryItem) async throws {
        throw AppError.localStorage("Fixture documents cannot be deleted.")
    }

    func localURL(for item: LibraryItem) async throws -> URL? {
        guard item.id == offlineItem.id, isValidDocument else {
            return nil
        }
        return documentURL
    }

    private var isValidDocument: Bool {
        guard let values = try? documentURL.resourceValues(
            forKeys: [
                .isRegularFileKey,
                .isSymbolicLinkKey,
                .fileSizeKey,
            ]) else {
            return false
        }
        return values.isRegularFile == true
            && values.isSymbolicLink != true
            && (values.fileSize ?? 0) > 0
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
