import SwiftUI

enum RootDestination: String, CaseIterable, Hashable, Identifiable {
    case library = "Library"
    case search = "Search"
    case downloads = "Downloads"
    case settings = "Settings"

    static func restored(from storedValue: String?) -> RootDestination {
        RootDestination(rawValue: storedValue ?? "") ?? .library
    }

    var id: Self { self }

    var systemImage: String {
        switch self {
        case .library: "books.vertical"
        case .search: "magnifyingglass"
        case .downloads: "arrow.down.circle"
        case .settings: "gearshape"
        }
    }
}

struct RootView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @SceneStorage("root.destination") private var storedDestination = RootDestination.library.rawValue
    @SceneStorage("reader.lastLibraryID") private var lastOpenedLibraryID: String?
    @State private var libraryViewModel: LibraryViewModel
    @State private var accountViewModel: AccountViewModel
    @State private var downloadViewModel: DownloadViewModel

    init(environment: AppEnvironment) {
        _libraryViewModel = State(
            initialValue: LibraryViewModel(repository: environment.libraryRepository))
        _accountViewModel = State(
            initialValue: AccountViewModel(
                repository: environment.libraryRepository,
                credentialStore: environment.credentialStore,
                clientFactory: environment.clientFactory))
        _downloadViewModel = State(
            initialValue: DownloadViewModel(store: environment.downloadStore))
    }

    var body: some View {
        Group {
            if horizontalSizeClass == .compact {
                compactNavigation
            } else {
                regularNavigation
            }
        }
        .task {
            await downloadViewModel.restore()
            guard let itemID = lastOpenedLibraryID else {
                return
            }
            await downloadViewModel.restoreReader(itemID: itemID)
            if downloadViewModel.readerSelection == nil {
                lastOpenedLibraryID = nil
            }
        }
        .onChange(of: downloadViewModel.readerSelection) { _, selection in
            if let selection {
                lastOpenedLibraryID = selection.item.id
            }
        }
    }

    private var compactNavigation: some View {
        TabView(selection: selectedDestination) {
            ForEach(RootDestination.allCases) { destination in
                NavigationStack {
                    destinationView(destination)
                }
                .tabItem {
                    Label(destination.rawValue, systemImage: destination.systemImage)
                }
                .accessibilityIdentifier(destination.accessibilityIdentifier)
                .tag(destination)
            }
        }
    }

    private var regularNavigation: some View {
        NavigationSplitView {
            List {
                ForEach(RootDestination.allCases) { destination in
                    Button {
                        storedDestination = destination.rawValue
                    } label: {
                        Label(destination.rawValue, systemImage: destination.systemImage)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityIdentifier(destination.accessibilityIdentifier)
                    .accessibilityAddTraits(
                        currentDestination == destination ? .isSelected : [])
                }
            }
            .navigationTitle("HAYA-TAB")
        } detail: {
            NavigationStack {
                destinationView(currentDestination)
            }
        }
    }

    private var currentDestination: RootDestination {
        RootDestination.restored(from: storedDestination)
    }

    private var selectedDestination: Binding<RootDestination> {
        Binding(
            get: { currentDestination },
            set: { storedDestination = $0.rawValue })
    }

    @ViewBuilder
    private func destinationView(_ destination: RootDestination) -> some View {
        switch destination {
        case .library:
            LibraryView(
                viewModel: libraryViewModel,
                downloadViewModel: downloadViewModel)
        case .search:
            SearchView()
        case .downloads:
            DownloadsView(viewModel: downloadViewModel)
        case .settings:
            SettingsView(viewModel: accountViewModel)
        }
    }
}

private extension RootDestination {
    var accessibilityIdentifier: String {
        "root.\(rawValue.lowercased())"
    }
}
