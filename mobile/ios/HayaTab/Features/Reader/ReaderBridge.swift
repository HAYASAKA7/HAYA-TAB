import Foundation

enum ReaderCommand: Equatable, Sendable {
    case load(Data)
    case playPause
    case stop
    case setTempo(Int)
}

enum ReaderEvent: Equatable, Sendable {
    case ready
    case loaded
    case failed(String)
}

enum ReaderBridgeError: Error, Equatable, Sendable {
    case unsupportedVersion
    case unknownMessage
    case invalidPayload
    case payloadTooLarge
}

enum ReaderBridgeContract {
    static let version = 1

    static func encode(
        _ command: ReaderCommand,
        maximumDocumentBytes: Int
    ) throws -> Data {
        var object: [String: Any] = ["version": version]

        switch command {
        case let .load(data):
            guard data.count <= maximumDocumentBytes else {
                throw ReaderBridgeError.payloadTooLarge
            }
            guard !data.isEmpty else {
                throw ReaderBridgeError.invalidPayload
            }
            object["type"] = "load"
            object["payload"] = ["base64": data.base64EncodedString()]
        case .playPause:
            object["type"] = "playPause"
        case .stop:
            object["type"] = "stop"
        case let .setTempo(beatsPerMinute):
            guard (30 ... 300).contains(beatsPerMinute) else {
                throw ReaderBridgeError.invalidPayload
            }
            object["type"] = "setTempo"
            object["payload"] = ["beatsPerMinute": beatsPerMinute]
        }

        guard JSONSerialization.isValidJSONObject(object) else {
            throw ReaderBridgeError.invalidPayload
        }
        return try JSONSerialization.data(
            withJSONObject: object,
            options: [.sortedKeys])
    }

    static func event(from body: Any) throws -> ReaderEvent {
        guard let object = body as? [String: Any],
              let messageVersion = object["version"] as? Int else {
            throw ReaderBridgeError.invalidPayload
        }
        guard messageVersion == version else {
            throw ReaderBridgeError.unsupportedVersion
        }
        guard let type = object["type"] as? String else {
            throw ReaderBridgeError.invalidPayload
        }

        switch type {
        case "ready":
            return .ready
        case "loaded":
            return .loaded
        case "error":
            let payload = object["payload"] as? [String: Any]
            let code = payload?["code"] as? String
            let allowedCodes = Set(["viewer_error"])
            return .failed(
                code.flatMap { allowedCodes.contains($0) ? $0 : nil }
                    ?? "viewer_error")
        default:
            throw ReaderBridgeError.unknownMessage
        }
    }

    static func commandObject(
        _ command: ReaderCommand,
        maximumDocumentBytes: Int
    ) throws -> [String: Any] {
        let data = try encode(
            command,
            maximumDocumentBytes: maximumDocumentBytes)
        guard let object = try JSONSerialization.jsonObject(with: data)
            as? [String: Any] else {
            throw ReaderBridgeError.invalidPayload
        }
        return object
    }
}
