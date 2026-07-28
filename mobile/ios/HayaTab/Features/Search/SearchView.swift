import SwiftUI

struct SearchView: View {
    @State private var query = ""

    var body: some View {
        ContentUnavailableView(
            "Search your library",
            systemImage: "magnifyingglass",
            description: Text("Find documents by title, artist, album, or category."))
            .navigationTitle("Search")
            .searchable(text: $query, prompt: "Title, artist, or category")
    }
}
