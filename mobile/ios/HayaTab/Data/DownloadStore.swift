import Foundation

protocol DownloadStoring: Sendable {
    func download(_ item: LibraryItem) async throws -> URL
    func offlineItems() async throws -> [LibraryItem]
    func delete(_ item: LibraryItem) async throws
}

actor DownloadStore: DownloadStoring {
    private let rootURL: URL
    private let downloader: any DocumentDownloading
    private let libraryStore: LibraryStore

    init(
        rootURL: URL,
        downloader: any DocumentDownloading,
        libraryStore: LibraryStore
    ) {
        self.rootURL = rootURL.standardizedFileURL
        self.downloader = downloader
        self.libraryStore = libraryStore
    }

    func download(_ item: LibraryItem) async throws -> URL {
        let filename = try stableFilename(for: item)
        let destinationURL = try childURL(filename)
        try prepareRoot()
        try removeStalePartials(for: item.id)

        let partialURL = try childURL(
            "\(item.id).\(UUID().uuidString.lowercased()).partial")
        defer {
            try? FileManager.default.removeItem(at: partialURL)
        }

        let transfer: DownloadTransfer
        do {
            transfer = try await downloader.download(
                path: item.relativePath,
                to: partialURL)
            try Task.checkCancellation()
        } catch is CancellationError {
            throw CancellationError()
        } catch let error as AppError {
            throw error
        } catch let error as WebDAVError {
            throw mapWebDAVError(error)
        } catch {
            throw AppError.transport("The document could not be downloaded.")
        }

        try validatePartialFile(partialURL, transfer: transfer)
        try applyCompleteFileProtection(to: partialURL)
        try Task.checkCancellation()
        let backupURL = try promote(
            partialURL,
            to: destinationURL,
            preservingExisting: FileManager.default.fileExists(atPath: destinationURL.path))

        do {
            try await libraryStore.setLocalFilename(filename, forID: item.id)
            if let backupURL {
                try? FileManager.default.removeItem(at: backupURL)
            }
            return destinationURL
        } catch {
            try? FileManager.default.removeItem(at: destinationURL)
            if let backupURL {
                try? FileManager.default.moveItem(at: backupURL, to: destinationURL)
            }
            if let appError = error as? AppError {
                throw appError
            }
            throw AppError.localStorage(
                "The offline document state could not be saved.")
        }
    }

    func offlineItems() async throws -> [LibraryItem] {
        try prepareRoot()
        let items: [LibraryItem]
        do {
            items = try await libraryStore.all()
        } catch let error as AppError {
            throw error
        } catch {
            throw AppError.localStorage("Offline documents could not be restored.")
        }

        var available: [LibraryItem] = []
        for item in items {
            guard let filename = item.localFilename else {
                continue
            }
            guard filename == (try? stableFilename(for: item)),
                  let url = try? childURL(filename),
                  isValidRegularFile(at: url) else {
                try? await libraryStore.setLocalFilename(nil, forID: item.id)
                continue
            }
            available.append(item)
        }
        return available.sorted {
            $0.title.localizedStandardCompare($1.title) == .orderedAscending
        }
    }

    func delete(_ item: LibraryItem) async throws {
        guard let filename = item.localFilename,
              filename == (try? stableFilename(for: item)) else {
            try await libraryStore.setLocalFilename(nil, forID: item.id)
            return
        }
        let url = try childURL(filename)
        do {
            if FileManager.default.fileExists(atPath: url.path) {
                try FileManager.default.removeItem(at: url)
            }
            try await libraryStore.setLocalFilename(nil, forID: item.id)
        } catch let error as AppError {
            throw error
        } catch {
            throw AppError.localStorage("The offline document could not be deleted.")
        }
    }

    private func prepareRoot() throws {
        do {
            try FileManager.default.createDirectory(
                at: rootURL,
                withIntermediateDirectories: true,
                attributes: [.protectionKey: FileProtectionType.complete])
        } catch {
            throw AppError.localStorage(
                "The offline documents directory could not be created.")
        }
    }

    private func stableFilename(for item: LibraryItem) throws -> String {
        let identifier = item.id.lowercased()
        let isStableIdentifier = identifier.count == 64
            && identifier.unicodeScalars.allSatisfy {
                CharacterSet(charactersIn: "0123456789abcdef").contains($0)
            }
        guard isStableIdentifier else {
            throw AppError.localStorage("The document identifier is invalid.")
        }

        let fileExtension = (item.relativePath as NSString).pathExtension.lowercased()
        let allowedExtensions: Set<String>
        switch item.kind {
        case .pdf:
            allowedExtensions = ["pdf"]
        case .guitarPro:
            allowedExtensions = ["gp", "gp3", "gp4", "gp5", "gpx", "xml", "musicxml", "mxl"]
        }
        guard allowedExtensions.contains(fileExtension) else {
            throw AppError.unsupportedDocument(item.relativePath)
        }
        return "\(identifier).\(fileExtension)"
    }

    private func childURL(_ filename: String) throws -> URL {
        guard filename == (filename as NSString).lastPathComponent,
              filename != ".",
              filename != ".." else {
            throw AppError.localStorage("The offline filename is invalid.")
        }
        let url = rootURL.appendingPathComponent(filename, isDirectory: false)
            .standardizedFileURL
        guard url.deletingLastPathComponent() == rootURL else {
            throw AppError.localStorage("The offline filename escaped its directory.")
        }
        return url
    }

    private func removeStalePartials(for itemID: String) throws {
        let files: [URL]
        do {
            files = try FileManager.default.contentsOfDirectory(
                at: rootURL,
                includingPropertiesForKeys: nil,
                options: [.skipsHiddenFiles])
        } catch {
            throw AppError.localStorage("Stale downloads could not be inspected.")
        }
        for file in files
        where file.lastPathComponent.hasPrefix("\(itemID).")
            && file.pathExtension == "partial" {
            do {
                try FileManager.default.removeItem(at: file)
            } catch {
                throw AppError.localStorage("A stale partial download could not be removed.")
            }
        }
    }

    private func validatePartialFile(
        _ url: URL,
        transfer: DownloadTransfer
    ) throws {
        guard isValidRegularFile(at: url) else {
            throw AppError.downloadIntegrity
        }
        let values: URLResourceValues
        do {
            values = try url.resourceValues(forKeys: [.fileSizeKey])
        } catch {
            throw AppError.downloadIntegrity
        }
        let actualByteCount = Int64(values.fileSize ?? 0)
        guard actualByteCount > 0 else {
            throw AppError.downloadIntegrity
        }
        if let expectedByteCount = transfer.expectedByteCount,
           expectedByteCount >= 0,
           actualByteCount != expectedByteCount {
            throw AppError.downloadIntegrity
        }
        if let validatorETag = transfer.validatorETag,
           transfer.responseETag != validatorETag {
            throw AppError.remoteChanged
        }
    }

    private func isValidRegularFile(at url: URL) -> Bool {
        guard let values = try? url.resourceValues(
            forKeys: [.isRegularFileKey, .isSymbolicLinkKey, .fileSizeKey]) else {
            return false
        }
        return values.isRegularFile == true
            && values.isSymbolicLink != true
            && (values.fileSize ?? 0) > 0
    }

    private func applyCompleteFileProtection(to url: URL) throws {
        do {
            try FileManager.default.setAttributes(
                [.protectionKey: FileProtectionType.complete],
                ofItemAtPath: url.path)
        } catch {
            throw AppError.localStorage(
                "Complete file protection could not be applied.")
        }
    }

    private func promote(
        _ partialURL: URL,
        to destinationURL: URL,
        preservingExisting: Bool
    ) throws -> URL? {
        do {
            guard preservingExisting else {
                try FileManager.default.moveItem(at: partialURL, to: destinationURL)
                return nil
            }

            let backupName = "\(destinationURL.lastPathComponent).\(UUID().uuidString).backup"
            _ = try FileManager.default.replaceItemAt(
                destinationURL,
                withItemAt: partialURL,
                backupItemName: backupName,
                options: [.withoutDeletingBackupItem])
            return rootURL.appendingPathComponent(backupName)
        } catch {
            throw AppError.localStorage(
                "The verified download could not replace the offline document.")
        }
    }

    private func mapWebDAVError(_ error: WebDAVError) -> AppError {
        switch error {
        case .authenticationRequired:
            .authentication
        case .httpStatus(412):
            .remoteChanged
        case .insecureTransport:
            .transport("The cloud account must use HTTPS.")
        case .remoteNotFound, .invalidRemotePath, .invalidResponse, .httpStatus, .transport:
            .transport("The document could not be downloaded.")
        }
    }
}
