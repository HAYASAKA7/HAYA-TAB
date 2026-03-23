# WebDAV Integration Guide

HAYA-TAB supports WebDAV integration, allowing you to sync your sheet music library with a cloud storage provider (like Nextcloud, ownCloud, Baidu Cloud, jianguoyun, or any standard WebDAV server).

## Enabling WebDAV

1.  Open **Settings** in the HAYA-TAB application.
2.  Scroll down to the **WebDAV** section.
3.  Check the box labeled **Enable WebDAV**.

## Configuration

If you are enabling WebDAV for the first time, a configuration window will appear. You can also edit these settings later by clicking the "Edit" button (pencil icon) next to the WebDAV address.

Enter the following details:

*   **URL**: The full WebDAV URL of your server (e.g., `https://cloud.example.com/remote.php/webdav/` for Nextcloud).
*   **Username**: Your WebDAV username.
*   **Password**: Your WebDAV password or app password.

Click **Test Connection** to verify your settings. If successful, click **Save**.

## Multi-Device Sync with Volumes (v2.3.0+)

HAYA-TAB v2.3.0 introduces a **volume system** for seamless multi-device synchronization:

### Volume System Overview

Each cloud drive (or directory served by your WebDAV server) is treated as a **volume**. Each volume:

- Has a hidden metadata directory (`haya-metadata/`) that identifies and tracks the volume
- Contains metadata about all files uploaded via HAYA-TAB to that volume (including Title, Artist, Album, and Categories)
- Enables multiple devices to discover and sync with the same cloud drive, including automatic reconstruction of category associations
- Tracks upload history, metadata, and source device information
- **CloudPath Persistence (v2.3.9+)**: Maintains the remote file mapping (`CloudPath`) even after a cloud tab is downloaded to local storage, ensuring subsequent metadata or category changes can still be synchronized back to the cloud fingerprint.

### How It Works

1. **Volume Discovery**
   - When you enable WebDAV or reconnect, HAYA-TAB automatically scans your WebDAV server
   - It discovers all existing volumes (directories with fingerprint files)
   - It automatically creates fingerprint files for new directories
   - Each device can independently discover the same volumes

2. **Fingerprint Files**
   - Each volume root directory contains a hidden metadata directory: `haya-metadata/`
   - This metadata directory contains:
     - Volume ID (unique identifier for the drive)
     - Volume name (e.g., "Google Drive", "OneDrive")
     - Creation date and last seen date
     - List of all files uploaded via HAYA-TAB, with metadata (title, artist, album, type, upload date, source device)
   - Fingerprint buckets are stored under:
     - `haya-metadata/bucket-00.json` ... `haya-metadata/bucket-15.json`

3. **PDF Annotation Sync (v2.4.5+)**
   - PDF annotations are synced as lightweight JSON files (non-destructive; original PDF is never modified).
   - Annotation files are stored inside the same hidden metadata directory:
     - `haya-metadata/annotations/<relative_file_path>.p<page>.json`
   - Example:
     - PDF relative path: `scores/etude.pdf`
     - Page: `3`
     - Remote annotation JSON path: `haya-metadata/annotations/scores/etude.pdf.p3.json`
   - This avoids creating extra visible directories in user content areas and keeps path mapping stable across devices.

4. **Multi-Device Workflow**
   ```
   Device A: Uploads file.gp5 to /gdrive/ → Fingerprint updated
                           ↓
   Device B: Connects to WebDAV → Discovers /gdrive/ volume
             Reads fingerprint → Syncs file automatically
             Now knows about file.gp5 from Device A
   ```

5. **Metadata Persistence**
   - When you upload a file, its metadata (title, artist, album) is saved in the fingerprint
   - When another device discovers the same volume, it automatically imports files with their metadata
   - This ensures consistent metadata across all devices without re-parsing files

### Automatic Features

- **Auto-Initialization**: On app startup, WebDAV volumes are automatically discovered and initialized
- **Connection Monitoring**: The app continuously monitors your WebDAV connection
- **Auto-Reconnection**: If connection is lost and then restored, the volume system reinitializes automatically
- **Health Checks**: The app periodically checks if all registered volumes are still accessible
- **Legacy Migration**: Existing cloud tabs are automatically migrated to use the volume system

## Using Cloud Library

Once enabled, you can access your cloud files directly from the library view.

### Downloading Files

1.  Go to the **Library** view.
2.  Ensure you are in **Singles** mode.
3.  Click the **Cloud Library** button (cloud icon) in the top header.
4.  Browse your remote files, select the ones you want to download, and click **Download Selected**.
   - You can browse across all discovered volumes
   - Files show their metadata in the listing

### Uploading Files

1.  Select one or more tabs in the **Library** view (using Select Mode).
2.  In the batch action bar at the bottom, click **Upload**.
3.  Select the destination folder on your WebDAV server and click **Upload**.
   - Choose any volume or subdirectory
   - Upload automatically creates fingerprint records
   - Metadata is preserved in the fingerprint

Alternatively, you can right-click any single tab and select **Upload to Cloud**.

### Viewing Across Multiple Volumes

Once you have files in different cloud volumes:

1. The **Cloud Library** view shows files from all accessible volumes
2. Files are organized by volume in the file browser
3. You can download or view files from any volume seamlessly
4. If a volume becomes unavailable, files from that volume show as offline but remain in your library

## Troubleshooting

### "Connection Lost" Message

- The app continuously checks your WebDAV connection
- If you see this message, check:
  - Your network connection
  - WebDAV server status
  - Username/password (credentials may have expired)

### Orphaned Tabs

In rare cases, if a volume is deleted or becomes inaccessible:

- Tabs associated with that volume are marked as "orphaned"
- The app will notify you and offer cleanup options
- You can choose to keep or delete orphaned tabs

### Volume Not Discovered

If a volume doesn't appear:

1. Open Settings and click "Test Connection" to verify connectivity
2. Try disconnecting and reconnecting WebDAV
3. Check that the volume directory is accessible with your credentials
4. Ensure you have read permissions on the WebDAV root

## Advanced: Manual Volume Management

For advanced users, you can manually manage volumes:

- **View Registered Volumes**: Go to Settings → WebDAV → View Volumes (if available)
- **Create New Volume**: Use the "Create Volume" option to manually organize your cloud storage
- **Remove Volume**: Delete a volume to remove all associated tabs from your library
- **Refresh Volumes**: Manually trigger volume discovery by reconnecting WebDAV
