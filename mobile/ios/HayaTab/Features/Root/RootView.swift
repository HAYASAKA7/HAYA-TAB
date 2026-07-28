import SwiftUI

enum RootDestination: String, CaseIterable, Hashable, Identifiable {
    case library = "Library"
    case search = "Search"
    case downloads = "Downloads"
    case settings = "Settings"

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
    @State private var compactSelection: RootDestination = .library
    @State private var splitSelection: RootDestination? = .library
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
        if horizontalSizeClass == .compact {
            compactNavigation
        } else {
            regularNavigation
        }
    }

    private var compactNavigation: some View {
        TabView(selection: $compactSelection) {
            ForEach(RootDestination.allCases) { destination in
                NavigationStack {
                    destinationView(destination)
                }
                .tabItem {
                    Label(destination.rawValue, systemImage: destination.systemImage)
                }
                .tag(destination)
            }
        }
    }

    private var regularNavigation: some View {
        NavigationSplitView {
            List(RootDestination.allCases, selection: $splitSelection) { destination in
                Label(destination.rawValue, systemImage: destination.systemImage)
                    .tag(destination)
            }
            .navigationTitle("HAYA-TAB")
        } detail: {
            NavigationStack {
                destinationView(splitSelection ?? .library)
            }
        }
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
