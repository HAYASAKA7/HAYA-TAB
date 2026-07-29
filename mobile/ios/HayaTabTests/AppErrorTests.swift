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

    func testAssociatedDetailsNeverAppearInUserFacingPresentation() {
        let marker = "Basic test-only-secret"
        let errors: [AppError] = [
            .transport(marker),
            .unsafeRemotePath(marker),
            .unsupportedDocument(marker),
            .localStorage(marker),
        ]

        for error in errors {
            XCTAssertFalse(error.title.contains(marker))
            XCTAssertFalse(error.recoverySuggestion.contains(marker))
        }
    }

    func testEveryErrorCodeIsStableAndUnique() {
        let errors = AppError.presentationFixtures
        let codes = errors.map(\.code)

        XCTAssertEqual(codes.count, Set(codes).count)
        XCTAssertTrue(codes.allSatisfy { !$0.isEmpty })
        XCTAssertTrue(errors.allSatisfy { !$0.title.isEmpty })
        XCTAssertTrue(errors.allSatisfy { !$0.recoverySuggestion.isEmpty })
    }
}
