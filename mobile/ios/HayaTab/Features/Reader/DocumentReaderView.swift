import SwiftUI

struct DocumentReaderView: View {
    let item: LibraryItem
    let documentURL: URL
    @State private var session = ReaderSession()

    var body: some View {
        ReaderWebView(
            documentURL: documentURL,
            kind: item.kind,
            session: session)
            .overlay {
                if let error = session.error {
                    ContentUnavailableView {
                        Label(error.title, systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error.recoverySuggestion)
                    }
                    .background(.regularMaterial)
                }
            }
            .navigationTitle(item.title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                if item.kind == .guitarPro {
                    ToolbarItemGroup(placement: .bottomBar) {
                        Button("Play or Pause", systemImage: "playpause") {
                            session.send(.playPause)
                        }
                        .disabled(!session.isLoaded)

                        Button("Stop", systemImage: "stop.fill") {
                            session.send(.stop)
                        }
                        .disabled(!session.isLoaded)

                        Menu("Tempo", systemImage: "metronome") {
                            ForEach([60, 80, 100, 120, 140, 160], id: \.self) { tempo in
                                Button("\(tempo) BPM") {
                                    session.send(.setTempo(tempo))
                                }
                            }
                        }
                        .disabled(!session.isLoaded)
                    }
                }
            }
    }
}
