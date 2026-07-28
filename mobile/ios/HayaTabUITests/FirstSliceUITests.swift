import XCTest

final class FirstSliceUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testCompactRootDestinationsExistAndDesktopOnlySettingsStayHidden() throws {
        guard !isIPadSimulator else {
            throw XCTSkip("Compact navigation is verified on an iPhone simulator")
        }

        let app = launchApp()

        XCTAssertTrue(app.tabBars.buttons["Library"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.tabBars.buttons["Search"].exists)
        XCTAssertTrue(app.tabBars.buttons["Downloads"].exists)
        XCTAssertTrue(app.tabBars.buttons["Settings"].exists)
        XCTAssertFalse(app.staticTexts["Keyboard shortcuts"].exists)
    }

    func testRegularRootDestinationsExistAndDesktopOnlySettingsStayHidden() throws {
        guard isIPadSimulator else {
            throw XCTSkip("Regular navigation is verified on an iPad simulator")
        }

        let app = launchApp()

        for label in ["Library", "Search", "Downloads", "Settings"] {
            XCTAssertTrue(app.staticTexts[label].waitForExistence(timeout: 5))
        }
        XCTAssertFalse(app.staticTexts["Keyboard shortcuts"].exists)
    }

    private var isIPadSimulator: Bool {
        ProcessInfo.processInfo.environment["SIMULATOR_DEVICE_NAME"]?.hasPrefix("iPad") == true
    }

    private func launchApp() -> XCUIApplication {
        let app = XCUIApplication()
        app.launchArguments = ["-use-fixture-library"]
        app.launch()
        return app
    }
}
