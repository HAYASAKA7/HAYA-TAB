import Foundation
import XCTest
@testable import HayaTab

@MainActor
final class ReaderBridgeTests: XCTestCase {
    func testLoadCommandEncodesVersionedBase64WithoutRawBytes() throws {
        let encoded = try ReaderBridgeContract.encode(
            .load(Data([0x00, 0x01, 0xfe, 0xff])),
            maximumDocumentBytes: 4)
        let object = try XCTUnwrap(
            JSONSerialization.jsonObject(with: encoded) as? [String: Any])
        let payload = try XCTUnwrap(object["payload"] as? [String: Any])

        XCTAssertEqual(object["version"] as? Int, 1)
        XCTAssertEqual(object["type"] as? String, "load")
        XCTAssertEqual(payload["base64"] as? String, "AAH+/w==")
        XCTAssertFalse(String(data: encoded, encoding: .utf8)?.contains("password") == true)
    }

    func testOversizedLoadCommandIsRejectedBeforeEncoding() {
        XCTAssertThrowsError(
            try ReaderBridgeContract.encode(
                .load(Data(repeating: 0x41, count: 5)),
                maximumDocumentBytes: 4)
        ) { error in
            XCTAssertEqual(error as? ReaderBridgeError, .payloadTooLarge)
        }
    }

    func testPlaybackCommandsAreAllowListedAndValidated() throws {
        let commands: [(ReaderCommand, String)] = [
            (.playPause, "playPause"),
            (.stop, "stop"),
            (.setTempo(120), "setTempo"),
        ]

        for (command, expectedType) in commands {
            let encoded = try ReaderBridgeContract.encode(
                command,
                maximumDocumentBytes: 0)
            let object = try XCTUnwrap(
                JSONSerialization.jsonObject(with: encoded) as? [String: Any])
            XCTAssertEqual(object["type"] as? String, expectedType)
        }

        XCTAssertThrowsError(
            try ReaderBridgeContract.encode(
                .setTempo(301),
                maximumDocumentBytes: 0)
        ) { error in
            XCTAssertEqual(error as? ReaderBridgeError, .invalidPayload)
        }
    }

    func testReadyLoadedAndSanitizedErrorEventsDecode() throws {
        XCTAssertEqual(
            try ReaderBridgeContract.event(
                from: ["version": 1, "type": "ready"]),
            .ready)
        XCTAssertEqual(
            try ReaderBridgeContract.event(
                from: ["version": 1, "type": "loaded"]),
            .loaded)
        XCTAssertEqual(
            try ReaderBridgeContract.event(
                from: [
                    "version": 1,
                    "type": "error",
                    "payload": ["code": "viewer_error"],
                ]),
            .failed("viewer_error"))
    }

    func testUnknownAndUnsupportedMessagesAreRejected() {
        XCTAssertThrowsError(
            try ReaderBridgeContract.event(
                from: ["version": 2, "type": "ready"])
        ) { error in
            XCTAssertEqual(error as? ReaderBridgeError, .unsupportedVersion)
        }
        XCTAssertThrowsError(
            try ReaderBridgeContract.event(
                from: ["version": 1, "type": "documentBytes"])
        ) { error in
            XCTAssertEqual(error as? ReaderBridgeError, .unknownMessage)
        }
    }
}
