import SwiftUI

struct LibraryView: View {
    let viewModel: LibraryViewModel

    var body: some View {
        Group {
            switch viewModel.state {
            case .idle, .loading:
                ProgressView("Loading library…")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            case .loaded where viewModel.items.isEmpty:
                ContentUnavailableView(
                    "No documents yet",
                    systemImage: "books.vertical",
                    description: Text("Connect your cloud library in Settings."))
            case .loaded:
                List(viewModel.items) { item in
                    LibraryRow(item: item)
                }
            case let .failed(error):
                ContentUnavailableView {
                    Label(error.title, systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error.recoverySuggestion)
                } actions: {
                    if error.isRetryable {
                        Button("Try Again") {
                            Task { await viewModel.load() }
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
