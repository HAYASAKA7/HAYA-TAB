import Foundation
import SwiftData

@Model
final class LibraryRecord {
    @Attribute(.unique) var id: String
    var volumeID: String
    var relativePath: String
    var title: String
    var artist: String
    var album: String
    var kindRawValue: String
    var categoriesData: Data
    var localFilename: String?
    var remoteRevision: String?

    init(item: LibraryItem) {
        id = item.id
        volumeID = item.volumeID
        relativePath = item.relativePath
        title = item.title
        artist = item.artist
        album = item.album
        kindRawValue = item.kind.rawValue
        categoriesData = (try? JSONEncoder().encode(item.categories)) ?? Data("[]".utf8)
        localFilename = item.localFilename
        remoteRevision = item.remoteRevision
    }

    func update(from item: LibraryItem) {
        volumeID = item.volumeID
        relativePath = item.relativePath
        title = item.title
        artist = item.artist
        album = item.album
        kindRawValue = item.kind.rawValue
        categoriesData = (try? JSONEncoder().encode(item.categories)) ?? Data("[]".utf8)
        localFilename = item.localFilename ?? localFilename
        remoteRevision = item.remoteRevision
    }

    func domainValue() throws -> LibraryItem {
        guard let kind = DocumentKind(rawValue: kindRawValue) else {
            throw AppError.localStorage("Unknown document kind: \(kindRawValue)")
        }

        return LibraryItem(
            id: id,
            volumeID: volumeID,
            relativePath: relativePath,
            title: title,
            artist: artist,
            album: album,
            kind: kind,
            categories: try JSONDecoder().decode([String].self, from: categoriesData),
            localFilename: localFilename,
            remoteRevision: remoteRevision)
    }
}

actor LibraryStore {
    private let container: ModelContainer

    init(container: ModelContainer) {
        self.container = container
    }

    func all() throws -> [LibraryItem] {
        let context = ModelContext(container)
        return try context.fetch(FetchDescriptor<LibraryRecord>())
            .map { try $0.domainValue() }
            .sorted { $0.id < $1.id }
    }

    func replace(with items: [LibraryItem]) throws {
        let context = ModelContext(container)
        let existing = try context.fetch(FetchDescriptor<LibraryRecord>())
        var recordsByID = Dictionary(uniqueKeysWithValues: existing.map { ($0.id, $0) })
        let incomingIDs = Set(items.map(\.id))

        for item in items {
            if let record = recordsByID.removeValue(forKey: item.id) {
                record.update(from: item)
            } else {
                context.insert(LibraryRecord(item: item))
            }
        }

        for record in existing where !incomingIDs.contains(record.id) {
            context.delete(record)
        }

        if context.hasChanges {
            try context.save()
        }
    }
}
