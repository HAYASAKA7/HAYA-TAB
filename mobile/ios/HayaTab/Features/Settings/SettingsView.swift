import SwiftUI

struct SettingsView: View {
    var body: some View {
        Form {
            Section("Cloud Library") {
                Label("WebDAV account", systemImage: "externaldrive.connected.to.line.below")
                Label("Sync over Wi-Fi and cellular", systemImage: "arrow.triangle.2.circlepath")
            }

            Section("About") {
                LabeledContent("Application", value: "HAYA-TAB")
                LabeledContent("Platform", value: "iOS")
            }
        }
        .navigationTitle("Settings")
    }
}
