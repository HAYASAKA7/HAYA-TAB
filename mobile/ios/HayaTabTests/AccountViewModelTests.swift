import Foundation
import XCTest
@testable import HayaTab

@MainActor
final class AccountViewModelTests: XCTestCase {
    func testConnectTestsThenSavesThenRefreshes() async throws {
        let events = EventRecorder()
        let credentialStore = CredentialStoreSpy(events: events)
        let connection = ConnectionStub(events: events)
        let repository = AccountRepositoryStub(events: events)
        let factory = ClientFactoryStub(client: connection, events: events)
        let viewModel = AccountViewModel(
            repository: repository,
            credentialStore: credentialStore,
            clientFactory: factory)
        viewModel.serverURL = "https://CLOUD.example.test/music"
        viewModel.username = "  haya  "
        viewModel.password = "test-only-secret"

        await viewModel.connect()

        let saved = try XCTUnwrap(credentialStore.savedCredential())
        XCTAssertEqual(saved.baseURL.absoluteString, "https://cloud.example.test/music/")
        XCTAssertEqual(saved.username, "haya")
        XCTAssertEqual(saved.password, "test-only-secret")
        XCTAssertEqual(viewModel.state, .connected)
        XCTAssertEqual(events.snapshot(), ["make", "test", "save", "refresh"])
    }

    func testConnectionFailureDoesNotSaveOrRefresh() async {
        let events = EventRecorder()
        let credentialStore = CredentialStoreSpy(events: events)
        let connection = ConnectionStub(
            events: events,
            failure: .authenticationRequired)
        let repository = AccountRepositoryStub(events: events)
        let viewModel = AccountViewModel(
            repository: repository,
            credentialStore: credentialStore,
            clientFactory: ClientFactoryStub(client: connection, events: events))
        viewModel.serverURL = "https://cloud.example.test/music/"
        viewModel.username = "haya"
        viewModel.password = "wrong-secret"

        await viewModel.connect()

        XCTAssertNil(credentialStore.savedCredential())
        XCTAssertEqual(viewModel.state, .failed(.authentication))
        XCTAssertEqual(events.snapshot(), ["make", "test"])
    }

    func testHTTPServerIsRejectedBeforeClientCreation() async {
        let events = EventRecorder()
        let credentialStore = CredentialStoreSpy(events: events)
        let connection = ConnectionStub(events: events)
        let repository = AccountRepositoryStub(events: events)
        let viewModel = AccountViewModel(
            repository: repository,
            credentialStore: credentialStore,
            clientFactory: ClientFactoryStub(client: connection, events: events))
        viewModel.serverURL = "http://cloud.example.test/music/"
        viewModel.username = "haya"
        viewModel.password = "test-only-secret"

        await viewModel.connect()

        XCTAssertEqual(
            viewModel.state,
            .failed(.transport("The cloud account must use HTTPS.")))
        XCTAssertTrue(events.snapshot().isEmpty)
        XCTAssertNil(credentialStore.savedCredential())
    }

    func testBlankUsernameIsRejectedBeforeClientCreation() async {
        let events = EventRecorder()
        let viewModel = AccountViewModel(
            repository: AccountRepositoryStub(events: events),
            credentialStore: CredentialStoreSpy(events: events),
            clientFactory: ClientFactoryStub(
                client: ConnectionStub(events: events),
                events: events))
        viewModel.serverURL = "https://cloud.example.test/music/"
        viewModel.username = "   "
        viewModel.password = "test-only-secret"

        await viewModel.connect()

        XCTAssertEqual(viewModel.state, .failed(.authentication))
        XCTAssertTrue(events.snapshot().isEmpty)
    }

    func testEmptyPasswordIsRejectedBeforeClientCreation() async {
        let events = EventRecorder()
        let viewModel = AccountViewModel(
            repository: AccountRepositoryStub(events: events),
            credentialStore: CredentialStoreSpy(events: events),
            clientFactory: ClientFactoryStub(
                client: ConnectionStub(events: events),
                events: events))
        viewModel.serverURL = "https://cloud.example.test/music/"
        viewModel.username = "haya"
        viewModel.password = ""

        await viewModel.connect()

        XCTAssertEqual(viewModel.state, .failed(.authentication))
        XCTAssertTrue(events.snapshot().isEmpty)
    }
}

private final class EventRecorder: @unchecked Sendable {
    private let lock = NSLock()
    private var events: [String] = []

    func record(_ event: String) {
        lock.lock()
        defer { lock.unlock() }
        events.append(event)
    }

    func snapshot() -> [String] {
        lock.lock()
        defer { lock.unlock() }
        return events
    }
}

private final class CredentialStoreSpy: CredentialStoring, @unchecked Sendable {
    private let events: EventRecorder
    private let lock = NSLock()
    private var credential: WebDAVCredential?

    init(events: EventRecorder) {
        self.events = events
    }

    func load() throws -> WebDAVCredential? {
        savedCredential()
    }

    func save(_ credential: WebDAVCredential) throws {
        events.record("save")
        lock.lock()
        defer { lock.unlock() }
        self.credential = credential
    }

    func delete() throws {
        lock.lock()
        defer { lock.unlock() }
        credential = nil
    }

    func savedCredential() -> WebDAVCredential? {
        lock.lock()
        defer { lock.unlock() }
        return credential
    }
}

private struct ClientFactoryStub: WebDAVClientBuilding {
    let client: any WebDAVServing
    let events: EventRecorder

    func makeClient(credential: WebDAVCredential) throws -> any WebDAVServing {
        events.record("make")
        return client
    }
}

private actor ConnectionStub: WebDAVServing {
    let events: EventRecorder
    let failure: WebDAVError?

    init(events: EventRecorder, failure: WebDAVError? = nil) {
        self.events = events
        self.failure = failure
    }

    func get(path: String) async throws -> WebDAVResponse {
        throw WebDAVError.remoteNotFound
    }

    func testConnection() async throws {
        events.record("test")
        if let failure {
            throw failure
        }
    }
}

private actor AccountRepositoryStub: LibraryRepositoryProtocol {
    let events: EventRecorder

    init(events: EventRecorder) {
        self.events = events
    }

    func cachedLibrary() async throws -> [HayaTab.LibraryItem] {
        []
    }

    func refresh() async throws -> [HayaTab.LibraryItem] {
        events.record("refresh")
        return []
    }
}
