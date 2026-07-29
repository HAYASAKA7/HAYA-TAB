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

        for identifier in [
            "root.library",
            "root.search",
            "root.downloads",
            "root.settings",
        ] {
            XCTAssertTrue(
                app.buttons[identifier].waitForExistence(timeout: 5),
                "Missing accessible root destination \(identifier)")
        }
        XCTAssertFalse(app.staticTexts["Keyboard shortcuts"].exists)
    }

    func testRegularRootDestinationsExistAndDesktopOnlySettingsStayHidden() throws {
        guard isIPadSimulator else {
            throw XCTSkip("Regular navigation is verified on an iPad simulator")
        }

        XCUIDevice.shared.orientation = .portrait
        let app = launchApp()

        for identifier in [
            "root.library",
            "root.search",
            "root.downloads",
            "root.settings",
        ] {
            XCTAssertTrue(
                app.buttons[identifier].waitForExistence(timeout: 5),
                "Missing accessible iPad destination \(identifier)")
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
        XCTAssertTrue(app.buttons["account.connect"].exists)
        XCTAssertFalse(app.staticTexts["Keyboard shortcuts"].exists)
    }

    func testRejectedAccountStaysInlineWithoutBlockingAlert() {
        let app = launchApp()
        openSettings(in: app)

        app.buttons["settings.configureAccount"].tap()
        XCTAssertTrue(
            app.buttons["account.connect"].waitForExistence(timeout: 5))
        app.buttons["account.connect"].tap()

        XCTAssertTrue(
            app.staticTexts["Couldn’t sign in"].waitForExistence(timeout: 5))
        XCTAssertFalse(app.alerts.firstMatch.exists)
    }

    func testAccessibilityXXXLKeepsPrimaryLibraryActionReachable() {
        let app = launchApp(additionalArguments: [
            "-UIPreferredContentSizeCategoryName",
            "UICTContentSizeCategoryAccessibilityExtraExtraExtraLarge",
        ])

        XCTAssertTrue(
            app.collectionViews["library.list"].waitForExistence(timeout: 5))
        XCTAssertGreaterThan(
            app.buttons.matching(
                NSPredicate(
                    format: "identifier BEGINSWITH %@",
                    "library.download.")).count,
            0)
    }

    func testReaderRestoresAfterApplicationRelaunch() {
        let app = launchApp()
        let openPrimer = app.buttons["Open Primer"]
        XCTAssertTrue(openPrimer.waitForExistence(timeout: 5))
        openPrimer.tap()
        XCTAssertTrue(app.navigationBars["Primer"].waitForExistence(timeout: 5))

        app.terminate()
        app.launch()

        XCTAssertTrue(
            app.navigationBars["Primer"].waitForExistence(timeout: 5))
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

    private func launchApp(
        additionalArguments: [String] = []
    ) -> XCUIApplication {
        let app = XCUIApplication()
        app.launchArguments = ["-use-fixture-library"] + additionalArguments
        app.launch()
        return app
    }

    private func openSettings(in app: XCUIApplication) {
        if isIPadSimulator {
            let settings = app.buttons["root.settings"]
            XCTAssertTrue(settings.waitForExistence(timeout: 5))
            settings.tap()
        } else {
            let settings = app.buttons["root.settings"]
            XCTAssertTrue(settings.waitForExistence(timeout: 5))
            settings.tap()
        }

        XCTAssertTrue(
            app.buttons["settings.configureAccount"].waitForExistence(timeout: 5))
    }
}
