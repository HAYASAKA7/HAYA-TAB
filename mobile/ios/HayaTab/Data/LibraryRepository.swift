import Foundation

protocol LibraryRepositoryProtocol: Sendable {
    func cachedLibrary() async throws -> [LibraryItem]
    func refresh() async throws -> [LibraryItem]
}

actor LibraryRepository: LibraryRepositoryProtocol {
    private let store: LibraryStore
    private let webDAV: any WebDAVServing

    init(store: LibraryStore, webDAV: any WebDAVServing) {
        self.store = store
        self.webDAV = webDAV
    }

    func cachedLibrary() async throws -> [LibraryItem] {
        do {
            return try await store.all()
        } catch let error as AppError {
            throw error
        } catch {
            throw AppError.localStorage("The cached library could not be read.")
        }
    }

    func refresh() async throws -> [LibraryItem] {
        let buckets = try await loadManifestBuckets()
        let metadata = try requireMetadata(from: buckets)
        let items = try makeLibraryItems(
            volumeID: metadata.volumeID,
            buckets: buckets)

        try Task.checkCancellation()
        do {
            try await store.replace(with: items)
        } catch is CancellationError {
            throw CancellationError()
        } catch let error as AppError {
            throw error
        } catch {
            throw AppError.localStorage("The refreshed library could not be saved.")
        }
        return items
    }

    private func loadManifestBuckets() async throws -> [FingerprintBucket] {
        do {
            try Task.checkCancellation()
            let bucketZero = try await fetchManifestBucket(number: 0, from: webDAV)
            guard let bucketZero else {
                throw AppError.malformedManifest
            }

            let dataBuckets = try await fetchDataBuckets()
            try Task.checkCancellation()
            return [bucketZero] + dataBuckets.sorted { $0.bucketNumber < $1.bucketNumber }
        } catch is CancellationError {
            throw CancellationError()
        } catch let error as AppError {
            throw error
        } catch let error as WebDAVError {
            throw mapWebDAVError(error)
        } catch {
            throw AppError.transport("The cloud library could not be refreshed.")
        }
    }

    private func fetchDataBuckets() async throws -> [FingerprintBucket] {
        let server = webDAV
        return try await withThrowingTaskGroup(
            of: FingerprintBucket?.self,
            returning: [FingerprintBucket].self
        ) { group in
            var nextBucket = 1
            let initialRequestCount = min(4, 15)

            for _ in 0 ..< initialRequestCount {
                let number = nextBucket
                nextBucket += 1
                group.addTask {
                    try await fetchManifestBucket(number: number, from: server)
                }
            }

            var buckets: [FingerprintBucket] = []
            while let result = try await group.next() {
                if let bucket = result {
                    buckets.append(bucket)
                }
                if nextBucket <= 15 {
                    let number = nextBucket
                    nextBucket += 1
                    group.addTask {
                        try await fetchManifestBucket(number: number, from: server)
                    }
                }
            }
            return buckets
        }
    }
}

private struct LibraryCandidate: Sendable {
    let bucketNumber: Int
    let file: FingerprintFile
}

private func fetchManifestBucket(
    number: Int,
    from webDAV: any WebDAVServing
) async throws -> FingerprintBucket? {
    do {
        let response = try await webDAV.get(path: bucketPath(number))
        switch response.statusCode {
        case 200 ..< 300:
            return try decodeBucket(response.data, expectedNumber: number)
        case 401:
            throw WebDAVError.authenticationRequired
        case 404 where number > 0:
            return nil
        case 404:
            throw WebDAVError.remoteNotFound
        default:
            throw WebDAVError.httpStatus(response.statusCode)
        }
    } catch WebDAVError.remoteNotFound where number > 0 {
        return nil
    }
}

private func decodeBucket(_ data: Data, expectedNumber: Int) throws -> FingerprintBucket {
    do {
        let bucket = try JSONDecoder().decode(FingerprintBucket.self, from: data)
        guard bucket.bucketNumber == expectedNumber else {
            throw AppError.malformedManifest
        }
        return bucket
    } catch is AppError {
        throw AppError.malformedManifest
    } catch {
        throw AppError.malformedManifest
    }
}

private func requireMetadata(from buckets: [FingerprintBucket]) throws -> FingerprintMetadata {
    guard let bucketZero = buckets.first(where: { $0.bucketNumber == 0 }),
          let metadata = bucketZero.metadata,
          !metadata.volumeID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
          metadata.bucketCount == 16 else {
        throw AppError.malformedManifest
    }
    return metadata
}

private func makeLibraryItems(
    volumeID: String,
    buckets: [FingerprintBucket]
) throws -> [LibraryItem] {
    var candidatesByIdentity: [String: LibraryCandidate] = [:]

    for bucket in buckets {
        for file in bucket.files {
            try validateRelativePath(file.relativePath)
            _ = try documentKind(for: file)

            let identity = "\(volumeID)\u{0}\(file.relativePath)"
            let candidate = LibraryCandidate(bucketNumber: bucket.bucketNumber, file: file)
            if let existing = candidatesByIdentity[identity] {
                if candidateIsPreferred(candidate, over: existing) {
                    candidatesByIdentity[identity] = candidate
                }
            } else {
                candidatesByIdentity[identity] = candidate
            }
        }
    }

    var items: [LibraryItem] = []
    items.reserveCapacity(candidatesByIdentity.count)
    for candidate in candidatesByIdentity.values {
        let file = candidate.file
        items.append(LibraryItem(
            id: LibraryItem.stableID(volumeID: volumeID, relativePath: file.relativePath),
            volumeID: volumeID,
            relativePath: file.relativePath,
            title: file.title,
            artist: file.artist,
            album: file.album,
            kind: try documentKind(for: file),
            categories: file.categories,
            localFilename: nil,
            remoteRevision: file.uploadedAt))
    }
    return items.sorted { $0.id < $1.id }
}

private func validateRelativePath(_ path: String) throws {
    guard !path.isEmpty,
          !path.hasPrefix("/"),
          !path.hasPrefix("//"),
          !path.contains("\\"),
          !path.contains("?"),
          !path.contains("#"),
          URLComponents(string: path)?.scheme == nil else {
        throw AppError.unsafeRemotePath(path)
    }

    for rawSegment in path.split(separator: "/", omittingEmptySubsequences: false) {
        let segment = String(rawSegment)
        guard !segment.isEmpty,
              let decoded = segment.removingPercentEncoding,
              decoded != ".",
              decoded != "..",
              !decoded.contains("/"),
              !decoded.contains("\\") else {
            throw AppError.unsafeRemotePath(path)
        }
    }
}

private func documentKind(for file: FingerprintFile) throws -> DocumentKind {
    let fileExtension = (file.relativePath as NSString).pathExtension.lowercased()
    let extensionKind: DocumentKind
    switch fileExtension {
    case "pdf":
        extensionKind = .pdf
    case "gp", "gp3", "gp4", "gp5", "gpx", "xml", "musicxml", "mxl":
        extensionKind = .guitarPro
    default:
        throw AppError.unsupportedDocument(file.relativePath)
    }

    guard let contractKind = try? DocumentKind(contractValue: file.type),
          contractKind == extensionKind else {
        throw AppError.unsupportedDocument(file.relativePath)
    }
    return extensionKind
}

private func candidateIsPreferred(
    _ candidate: LibraryCandidate,
    over existing: LibraryCandidate
) -> Bool {
    let formatter = ISO8601DateFormatter()
    let candidateDate = formatter.date(from: candidate.file.uploadedAt)
    let existingDate = formatter.date(from: existing.file.uploadedAt)

    if let candidateDate, let existingDate, candidateDate != existingDate {
        return candidateDate > existingDate
    }
    if candidate.file.uploadedAt != existing.file.uploadedAt {
        return candidate.file.uploadedAt > existing.file.uploadedAt
    }
    if candidate.bucketNumber != existing.bucketNumber {
        return candidate.bucketNumber < existing.bucketNumber
    }
    return stableSignature(candidate.file) < stableSignature(existing.file)
}

private func stableSignature(_ file: FingerprintFile) -> String {
    [
        file.relativePath,
        file.title,
        file.artist,
        file.album,
        file.type,
        file.categories.joined(separator: "\u{1f}"),
        file.uploadedAt,
        file.uploadedBy,
    ].joined(separator: "\u{0}")
}

private func bucketPath(_ number: Int) -> String {
    String(format: "haya-metadata/bucket-%02d.json", number)
}

private func mapWebDAVError(_ error: WebDAVError) -> AppError {
    switch error {
    case .authenticationRequired:
        return .authentication
    case .remoteNotFound:
        return .transport("The cloud library metadata was not found.")
    case .invalidRemotePath:
        return .unsafeRemotePath("haya-metadata")
    case .insecureTransport:
        return .transport("The cloud account must use HTTPS.")
    case .invalidResponse, .httpStatus, .transport:
        return .transport("The cloud library could not be refreshed.")
    }
}
