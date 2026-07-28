import SwiftUI

struct LibraryView: View {
    let viewModel: LibraryViewModel
    let downloadViewModel: DownloadViewModel

    var body: some View {
        Group {
            switch viewModel.state {
            case .idle, .loading:
                ProgressView("Loading library…")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            case .loaded:
                libraryList
            case let .offline(error):
                offlineLibrary(error)
            case let .failed(error):
                ContentUnavailableView {
                    Label(error.title, systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error.recoverySuggestion)
                } actions: {
                    if error.isRetryable {
                        Button("Try Again") {
                            Task { await viewModel.refresh() }
                        }
                    }
                }
            }
        }
        .navigationTitle("Library")
        .task {
            if viewModel.state == .idle {
                await viewModel.load()
            }
        }
    }

    private var libraryList: some View {
        List {
            if viewModel.items.isEmpty {
                ContentUnavailableView(
                    "No documents yet",
                    systemImage: "books.vertical",
                    description: Text("Connect your cloud library in Settings."))
                .listRowBackground(Color.clear)
            } else {
                ForEach(viewModel.items) { item in
                    DownloadableLibraryRow(item: item, viewModel: downloadViewModel)
                }
            }
        }
        .refreshable {
            await viewModel.refresh()
        }
    }

    private func offlineLibrary(_ error: AppError) -> some View {
        List {
            Section {
                Label {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Showing offline library")
                            .font(.headline)
                        Text(error.recoverySuggestion)
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                } icon: {
                    Image(systemName: "icloud.slash")
                        .foregroundStyle(.orange)
                }
                .accessibilityElement(children: .combine)
            }

            Section("Documents") {
                ForEach(viewModel.items) { item in
                    DownloadableLibraryRow(item: item, viewModel: downloadViewModel)
                }
            }
        }
        .refreshable {
            await viewModel.refresh()
        }
    }
}

private struct DownloadableLibraryRow: View {
    let item: LibraryItem
    let viewModel: DownloadViewModel

    var body: some View {
        HStack(spacing: 12) {
            LibraryRow(item: item)
            Spacer(minLength: 8)
            control
        }
    }

    @ViewBuilder
    private var control: some View {
        switch viewModel.state(for: item) {
        case .queued, .downloading:
            HStack(spacing: 10) {
                ProgressView()
                    .accessibilityLabel("Downloading \(item.title)")
                Button {
                    viewModel.cancel(item)
                } label: {
                    Image(systemName: "xmark.circle.fill")
                }
                .accessibilityLabel("Cancel download of \(item.title)")
                .buttonStyle(.borderless)
            }
        case .availableOffline:
            Image(systemName: "checkmark.circle.fill")
                .foregroundStyle(.green)
                .accessibilityLabel("\(item.title) is available offline")
        case .failed:
            Button {
                viewModel.start(item)
            } label: {
                Image(systemName: "arrow.clockwise.circle")
            }
            .accessibilityLabel("Retry download of \(item.title)")
            .buttonStyle(.borderless)
        case nil:
            Button {
                viewModel.start(item)
            } label: {
                Image(systemName: "arrow.down.circle")
            }
            .accessibilityLabel("Download \(item.title)")
            .buttonStyle(.borderless)
        }
    }
}

private struct LibraryRow: View {
    let item: LibraryItem

    var body: some View {
        Label {
            VStack(alignment: .leading, spacing: 3) {
                Text(item.title)
                    .font(.headline)
                Text(item.artist.isEmpty ? item.relativePath : item.artist)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
        } icon: {
            Image(systemName: item.kind == .pdf ? "doc.richtext" : "music.note.list")
                .foregroundStyle(.tint)
        }
        .accessibilityElement(children: .combine)
    }
}
