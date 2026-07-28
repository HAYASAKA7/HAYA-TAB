import SwiftUI

struct DownloadsView: View {
    @Bindable var viewModel: DownloadViewModel
    @State private var deletionCandidate: LibraryItem?

    var body: some View {
        Group {
            if viewModel.displayItems.isEmpty {
                ContentUnavailableView(
                    "No offline documents",
                    systemImage: "arrow.down.circle",
                    description: Text("Downloaded documents will appear here."))
            } else {
                List(viewModel.displayItems) { item in
                    HStack(spacing: 12) {
                        VStack(alignment: .leading, spacing: 3) {
                            Text(item.title)
                                .font(.headline)
                            Text(item.artist.isEmpty ? item.relativePath : item.artist)
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        statusView(for: item)
                    }
                    .contentShape(Rectangle())
                    .onTapGesture {
                        if viewModel.state(for: item) == .availableOffline {
                            Task { await viewModel.open(item) }
                        }
                    }
                    .swipeActions {
                        if viewModel.state(for: item) == .availableOffline {
                            Button("Remove", systemImage: "trash", role: .destructive) {
                                deletionCandidate = item
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("Downloads")
        .task {
            await viewModel.restore()
        }
        .navigationDestination(item: $viewModel.readerSelection) { selection in
            DocumentReaderView(
                item: selection.item,
                documentURL: selection.documentURL)
        }
        .confirmationDialog(
            "Remove Offline Copy?",
            isPresented: Binding(
                get: { deletionCandidate != nil },
                set: { if !$0 { deletionCandidate = nil } }),
            presenting: deletionCandidate
        ) { item in
            Button("Remove Download", role: .destructive) {
                Task {
                    await viewModel.delete(item)
                    deletionCandidate = nil
                }
            }
            Button("Cancel", role: .cancel) {
                deletionCandidate = nil
            }
        } message: { item in
            Text("“\(item.title)” will remain available in the cloud library.")
        }
    }

    @ViewBuilder
    private func statusView(for item: LibraryItem) -> some View {
        switch viewModel.state(for: item) {
        case .queued:
            Label("Queued", systemImage: "clock")
                .foregroundStyle(.secondary)
        case .downloading:
            HStack {
                ProgressView()
                Button("Cancel") {
                    viewModel.cancel(item)
                }
            }
        case .availableOffline:
            Label("Offline", systemImage: "checkmark.circle.fill")
                .foregroundStyle(.green)
        case let .failed(error):
            Button {
                viewModel.start(item)
            } label: {
                Label(error.title, systemImage: "arrow.clockwise")
            }
        case nil:
            EmptyView()
        }
    }
}
