# WebDAV Integration Guide

HAYA-TAB supports WebDAV integration, allowing you to sync your sheet music library with a cloud storage provider (like Nextcloud, ownCloud, or any standard WebDAV server).

## Enabling WebDAV

1.  Open **Settings** in the HAYA-TAB application.
2.  Scroll down to the **WebDAV** section.
3.  Check the box labeled **Enable WebDAV**.

## Configuration

If you are enabling WebDAV for the first time, a configuration window will appear. You can also edit these settings later by clicking the "Edit" button (pencil icon) next to the WebDAV address.

Enter the following details:

*   **URL**: The full WebDAV URL of your server (e.g., `https://cloud.example.com/remote.php/webdav/`).
*   **Username**: Your WebDAV username.
*   **Password**: Your WebDAV password or app password.

Click **Test Connection** to verify your settings. If successful, click **Save**.

## Using Cloud Library

Once enabled, you can access your cloud files directly from the library view.

### Downloading Files

1.  Go to the **Library** view.
2.  Ensure you are in **Singles** mode.
3.  Click the **Cloud Library** button (cloud icon) in the top header.
4.  Browse your remote files, select the ones you want to download, and click **Download Selected**.

### Uploading Files

1.  Select one or more tabs in the **Library** view (using Select Mode).
2.  In the batch action bar at the bottom, click **Upload**.
3.  Select the destination folder on your WebDAV server and click **Upload**.

Alternatively, you can right-click any single tab and select **Upload to Cloud**.
