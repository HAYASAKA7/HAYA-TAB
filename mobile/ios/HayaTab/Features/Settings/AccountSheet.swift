import Foundation
import Observation
import SwiftUI

enum AccountConnectionState: Equatable {
    case idle
    case testing
    case connected
    case failed(AppError)
}

@MainActor
@Observable
final class AccountViewModel {
    private let repository: any LibraryRepositoryProtocol
    private let credentialStore: any CredentialStoring
    private let clientFactory: any WebDAVClientBuilding

    var serverURL = ""
    var username = ""
    var password = ""
    private(set) var state: AccountConnectionState = .idle

    init(
        repository: any LibraryRepositoryProtocol,
        credentialStore: any CredentialStoring,
        clientFactory: any WebDAVClientBuilding
    ) {
        self.repository = repository
        self.credentialStore = credentialStore
        self.clientFactory = clientFactory
    }

    func connect() async {
        let credential: WebDAVCredential
        do {
            credential = try validatedCredential()
        } catch let error as AppError {
            state = .failed(error)
            return
        } catch {
            state = .failed(.transport("Enter a valid HTTPS WebDAV address."))
            return
        }

        state = .testing
        do {
            let client = try clientFactory.makeClient(credential: credential)
            try await client.testConnection()
            try Task.checkCancellation()
            try credentialStore.save(credential)
            _ = try await repository.refresh()
            try Task.checkCancellation()
            state = .connected
        } catch is CancellationError {
            state = .idle
        } catch let error as AppError {
            state = .failed(error)
        } catch let error as WebDAVError {
            state = .failed(mapWebDAVError(error))
        } catch {
            state = .failed(.transport("The cloud account could not be verified."))
        }
    }

    private func validatedCredential() throws -> WebDAVCredential {
        let rawURL = serverURL.trimmingCharacters(in: .whitespacesAndNewlines)
        guard var components = URLComponents(string: rawURL),
              components.scheme?.lowercased() == "https" else {
            throw AppError.transport("The cloud account must use HTTPS.")
        }
        guard components.host?.isEmpty == false,
              components.user == nil,
              components.password == nil,
              components.query == nil,
              components.fragment == nil else {
            throw AppError.transport("Enter a valid HTTPS WebDAV address.")
        }

        components.scheme = "https"
        components.host = components.host?.lowercased()
        if !components.percentEncodedPath.hasSuffix("/") {
            components.percentEncodedPath += "/"
        }
        guard let normalizedURL = components.url else {
            throw AppError.transport("Enter a valid HTTPS WebDAV address.")
        }

        let normalizedUsername = username.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedUsername.isEmpty, !password.isEmpty else {
            throw AppError.authentication
        }
        return WebDAVCredential(
            baseURL: normalizedURL,
            username: normalizedUsername,
            password: password)
    }

    private func mapWebDAVError(_ error: WebDAVError) -> AppError {
        switch error {
        case .authenticationRequired:
            .authentication
        case .insecureTransport:
            .transport("The cloud account must use HTTPS.")
        case .remoteNotFound, .invalidRemotePath, .invalidResponse, .httpStatus, .transport:
            .transport("The cloud account could not be verified.")
        }
    }
}

struct AccountSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Bindable var viewModel: AccountViewModel

    var body: some View {
        NavigationStack {
            Form {
                Section("WebDAV Server") {
                    TextField("Server URL", text: $viewModel.serverURL)
                        .textContentType(.URL)
                        .keyboardType(.URL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .accessibilityIdentifier("account.serverURL")
                    TextField("Username", text: $viewModel.username)
                        .textContentType(.username)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .accessibilityIdentifier("account.username")
                    SecureField("Password", text: $viewModel.password)
                        .textContentType(.password)
                        .accessibilityIdentifier("account.password")
                }

                if case let .failed(error) = viewModel.state {
                    Section {
                        Label {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(error.title)
                                    .font(.headline)
                                Text(error.recoverySuggestion)
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                        } icon: {
                            Image(systemName: "exclamationmark.triangle")
                                .foregroundStyle(.orange)
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityIdentifier("account.error")
                    }
                }

                Section {
                    Button {
                        Task {
                            await viewModel.connect()
                            if viewModel.state == .connected {
                                dismiss()
                            }
                        }
                    } label: {
                        Text("Test and Save")
                            .frame(maxWidth: .infinity)
                            .opacity(viewModel.state == .testing ? 0 : 1)
                    }
                    .overlay {
                        if viewModel.state == .testing {
                            ProgressView()
                        }
                    }
                    .disabled(viewModel.state == .testing)
                    .accessibilityElement(children: .ignore)
                    .accessibilityLabel(
                        viewModel.state == .testing
                            ? "Testing cloud account"
                            : "Test and Save")
                    .accessibilityIdentifier("account.connect")
                }
            }
            .navigationTitle("Cloud Account")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                }
            }
        }
    }
}
