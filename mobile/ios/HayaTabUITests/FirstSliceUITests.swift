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

    func testFixtureLibraryLaunchesFromCacheWithoutNetwork() {
        let app = launchApp()

        XCTAssertTrue(app.staticTexts["Etude"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Primer"].exists)
    }

    func testCloudAccountSheetUsesNativeMobileFields() {
        let app = launchApp()

        if isIPadSimulator {
            app.staticTexts["Settings"].tap()
        } else {
            app.tabBars.buttons["Settings"].tap()
        }

        let configureButton = app.buttons["Configure WebDAV"]
        XCTAssertTrue(configureButton.waitForExistence(timeout: 5))
        configureButton.tap()

        XCTAssertTrue(app.navigationBars["Cloud Account"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.textFields["Server URL"].exists)
        XCTAssertTrue(app.textFields["Username"].exists)
        XCTAssertTrue(app.secureTextFields["Password"].exists)
        XCTAssertTrue(app.buttons["Test and Save"].exists)
        XCTAssertFalse(app.staticTexts["Keyboard shortcuts"].exists)
    }

    func testDownloadFailureStaysInlineAndAppearsInDownloads() {
        let app = launchApp()
        let downloadButton = app.buttons["Download Etude"]
        XCTAssertTrue(downloadButton.waitForExistence(timeout: 5))

        downloadButton.tap()

        XCTAssertTrue(
            app.buttons["Retry download of Etude"].waitForExistence(timeout: 5))
        if isIPadSimulator {
            app.staticTexts["Downloads"].tap()
        } else {
            app.tabBars.buttons["Downloads"].tap()
        }
        XCTAssertTrue(app.staticTexts["Etude"].waitForExistence(timeout: 5))
        XCTAssertFalse(app.alerts.firstMatch.exists)
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
