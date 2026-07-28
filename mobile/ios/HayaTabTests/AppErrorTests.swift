import XCTest
@testable import HayaTab

final class AppErrorTests: XCTestCase {
    func testAuthenticationErrorProvidesActionableRecovery() {
        let error = AppError.authentication

        XCTAssertEqual(error.code, "authentication")
        XCTAssertEqual(error.title, "Couldn’t sign in")
        XCTAssertTrue(error.isRetryable)
        XCTAssertFalse(error.recoverySuggestion.isEmpty)
    }

    func testUnsafePathIsNotRetryable() {
        let error = AppError.unsafeRemotePath("../outside.pdf")

        XCTAssertEqual(error.code, "unsafe_remote_path")
        XCTAssertEqual(error.title, "Unsafe cloud path")
        XCTAssertFalse(error.isRetryable)
        XCTAssertTrue(error.recoverySuggestion.contains("library"))
    }
}
