import Foundation
import SwiftUI

@main
struct HayaTabApp: App {
    private let environment: AppEnvironment

    init() {
        environment = ProcessInfo.processInfo.arguments.contains("-use-fixture-library")
            ? .fixture()
            : .live()
    }

    var body: some Scene {
        WindowGroup {
            RootView(environment: environment)
        }
    }
}
