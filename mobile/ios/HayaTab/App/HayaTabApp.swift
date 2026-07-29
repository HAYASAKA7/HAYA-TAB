import Foundation
import SwiftUI

@main
struct HayaTabApp: App {
    private let environment: AppEnvironment

    init() {
        environment = ProcessInfo.processInfo.arguments.contains("-use-fixture-library")
            ? .fixture()
            : .live()
        if ProcessInfo.processInfo.arguments.contains("-reset-restoration-state") {
            UserDefaults.standard.removeObject(
                forKey: "reader.lastLibraryID.persisted")
        }
    }

    var body: some Scene {
        WindowGroup {
            RootView(environment: environment)
        }
    }
}
