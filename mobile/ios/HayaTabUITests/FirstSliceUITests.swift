import XCTest

final class FirstSliceUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testRootDestinationsExistAndDesktopOnlySettingsStayHidden() {
        let app = XCUIApplication()
        app.launchArguments = ["-use-fixture-library"]
        app.launch()

        XCTAssertTrue(app.tabBars.buttons["Library"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.tabBars.buttons["Search"].exists)
        XCTAssertTrue(app.tabBars.buttons["Downloads"].exists)
        XCTAssertTrue(app.tabBars.buttons["Settings"].exists)
        XCTAssertFalse(app.staticTexts["Keyboard shortcuts"].exists)
    }
}
