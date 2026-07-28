import CryptoKit
import Foundation

enum DocumentKind: String, Codable, Sendable {
    case pdf
    case guitarPro

    init(contractValue: String) throws {
        switch contractValue.lowercased() {
        case "pdf":
            self = .pdf
        case "gp", "gp3", "gp4", "gp5", "gpx", "xml", "musicxml", "mxl":
            self = .guitarPro
        default:
            throw DecodingError.dataCorrupted(
                .init(codingPath: [], debugDescription: "Unsupported document type: \(contractValue)"))
        }
    }
}

enum OfflineState: String, Codable, Sendable {
    case cloudOnly
    case downloading
    case availableOffline
    case failed
}

struct LibraryItem: Identifiable, Codable, Equatable, Hashable, Sendable {
    let id: String
    let volumeID: String
    let relativePath: String
    let title: String
    let artist: String
    let album: String
    let kind: DocumentKind
    let categories: [String]
    let localFilename: String?
    let remoteRevision: String?

    static func stableID(volumeID: String, relativePath: String) -> String {
        let input = Data("\(volumeID)\n\(relativePath)".utf8)
        return SHA256.hash(data: input)
            .map { String(format: "%02x", $0) }
            .joined()
    }
}

struct FingerprintMetadata: Decodable, Equatable, Sendable {
    let volumeID: String
    let volumeName: String
    let createdAt: String
    let appVersion: String
    let deviceName: String
    let lastUpdated: String
    let bucketCount: Int

    enum CodingKeys: String, CodingKey {
        case volumeID = "volume_id"
        case volumeName = "volume_name"
        case createdAt = "created_at"
        case appVersion = "app_version"
        case deviceName = "device_name"
        case lastUpdated = "last_updated"
        case bucketCount = "bucket_count"
    }
}

struct FingerprintFile: Decodable, Equatable, Sendable {
    let relativePath: String
    let title: String
    let artist: String
    let album: String
    let type: String
    let categories: [String]
    let uploadedAt: String
    let uploadedBy: String

    enum CodingKeys: String, CodingKey {
        case relativePath = "relative_path"
        case title
        case artist
        case album
        case type
        case categories
        case uploadedAt = "uploaded_at"
        case uploadedBy = "uploaded_by"
    }

    func libraryItem(volumeID: String) throws -> LibraryItem {
        LibraryItem(
            id: LibraryItem.stableID(volumeID: volumeID, relativePath: relativePath),
            volumeID: volumeID,
            relativePath: relativePath,
            title: title,
            artist: artist,
            album: album,
            kind: try DocumentKind(contractValue: type),
            categories: categories,
            localFilename: nil,
            remoteRevision: uploadedAt)
    }
}

struct FingerprintBucket: Decodable, Equatable, Sendable {
    let bucketNumber: Int
    let metadata: FingerprintMetadata?
    let files: [FingerprintFile]

    enum CodingKeys: String, CodingKey {
        case bucketNumber = "bucket_number"
        case metadata
        case files
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        metadata = try container.decodeIfPresent(FingerprintMetadata.self, forKey: .metadata)
        files = try container.decode([FingerprintFile].self, forKey: .files)

        if let number = try container.decodeIfPresent(Int.self, forKey: .bucketNumber) {
            guard (1 ... 15).contains(number), metadata == nil else {
                throw DecodingError.dataCorruptedError(
                    forKey: .bucketNumber,
                    in: container,
                    debugDescription: "Data bucket number must be between 1 and 15")
            }
            bucketNumber = number
        } else if metadata?.bucketCount == 16 {
            bucketNumber = 0
        } else {
            throw DecodingError.keyNotFound(
                CodingKeys.metadata,
                .init(codingPath: decoder.codingPath, debugDescription: "Bucket zero requires valid metadata"))
        }
    }
}
