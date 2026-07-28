import SwiftUI

struct DownloadsView: View {
    var body: some View {
        ContentUnavailableView(
            "No offline documents",
            systemImage: "arrow.down.circle",
            description: Text("Downloaded documents will appear here."))
            .navigationTitle("Downloads")
    }
}
