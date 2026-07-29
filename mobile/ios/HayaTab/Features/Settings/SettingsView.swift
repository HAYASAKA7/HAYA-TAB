import SwiftUI

struct SettingsView: View {
    let viewModel: AccountViewModel
    @State private var isShowingAccount = false

    var body: some View {
        Form {
            Section("Cloud Library") {
                Label("WebDAV account", systemImage: "externaldrive.connected.to.line.below")
                Button("Configure WebDAV") {
                    isShowingAccount = true
                }
                .accessibilityIdentifier("settings.configureAccount")
                Label("Sync over Wi-Fi and cellular", systemImage: "arrow.triangle.2.circlepath")
            }

            Section("About") {
                LabeledContent("Application", value: "HAYA-TAB")
                LabeledContent("Platform", value: "iOS")
            }
        }
        .navigationTitle("Settings")
        .sheet(isPresented: $isShowingAccount) {
            AccountSheet(viewModel: viewModel)
        }
    }
}
