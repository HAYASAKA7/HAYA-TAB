import SwiftUI

struct DownloadsView: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
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
                    downloadRow(for: item)
                    .contentShape(Rectangle())
                    .onTapGesture {
                        if viewModel.state(for: item) == .availableOffline {
                            Task { await viewModel.open(item) }
                        }
                    }
                }
                .accessibilityIdentifier("downloads.list")
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

    private func downloadRow(for item: LibraryItem) -> some View {
        Group {
            if dynamicTypeSize.isAccessibilitySize {
                VStack(alignment: .leading, spacing: 12) {
                    downloadDescription(for: item)
                    statusView(for: item)
                        .frame(maxWidth: .infinity, alignment: .trailing)
                }
            } else {
                HStack(spacing: 12) {
                    downloadDescription(for: item)
                    Spacer()
                    statusView(for: item)
                }
            }
        }
        .swipeActions {
            if viewModel.state(for: item) == .availableOffline {
                Button("Remove", systemImage: "trash", role: .destructive) {
                    deletionCandidate = item
                }
                .accessibilityIdentifier("downloads.remove.\(item.id)")
            }
        }
    }

    private func downloadDescription(for item: LibraryItem) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(item.title)
                .font(.headline)
            Text(item.artist.isEmpty ? item.relativePath : item.artist)
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .accessibilityElement(children: .combine)
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
                .accessibilityIdentifier("downloads.cancel.\(item.id)")
            }
        case .availableOffline:
            Label("Offline", systemImage: "checkmark.circle.fill")
                .foregroundStyle(.green)
                .accessibilityIdentifier("downloads.open.\(item.id)")
        case let .failed(error):
            Button {
                viewModel.start(item)
            } label: {
                Label(error.title, systemImage: "arrow.clockwise")
            }
            .accessibilityIdentifier("downloads.retry.\(item.id)")
        case nil:
            EmptyView()
        }
    }
}
