# HAYA-TAB Architecture

HAYA-TAB is a cross-platform desktop application built using the [Wails](https://wails.io/) framework, combining a Go backend with a modern web frontend (Vue 3 + Vite).

## System Overview

The system architecture is divided into two primary layers:
1. **Frontend (Presentation Layer):** A responsive, web-based UI rendering views, handling user interactions, and maintaining client-side state.
2. **Backend (Application Layer):** A native Go application managing the business logic, database operations, file system interactions, synchronization, and the Wails application lifecycle.

Communication between the frontend and backend happens via Wails' IPC (Inter-Process Communication) bridge.

## Frontend Architecture

- **Framework:** Vue 3 via Vite.
- **State Management:** Pinia stores (`src/stores`) manage application state including tabs, settings, UI states, and viewer configurations.
- **Routing:** Handled via Vue's dynamic component rendering or a minimal router setup depending on state (e.g., `HomeView` vs `LibraryView`).
- **Styling:** Custom CSS with a responsive grid layout.
- **Viewers:**
  - **PDF.js:** For rendering standard PDF documents.
  - **alphaTab:** A music notation and guitar tablature rendering engine for playing `.gp`, `.gp5`, `.gpx`, and MusicXML files.
- **Components:** Modular components organized into grids, layouts, modals, viewers, and common UI elements.

## Backend Architecture

The Go backend is structured into distinct, modular packages:

- **`internal/app`:** The core application logic. It initializes the Wails app lifecycle and exposes methods bound to the frontend (e.g., managing tabs, categories, file dialogs, settings, migrations, and WebDAV operations).
- **`pkg/store`:** The data persistence layer. Uses SQLite (via `modernc.org/sqlite`) for local storage, implementing Full-Text Search (FTS5) for fast querying of titles, artists, and albums.
- **`pkg/metadata`:** Parses metadata from filenames and tab files (like Guitar Pro). Includes MusicBrainz API integration (`musicbrainz.go`) for retrieving missing album artwork and information.
- **`pkg/coverpool`:** A worker pool for concurrent, high-performance downloading of cover images without blocking the main application flow.
- **`pkg/sync` & `pkg/watcher`:** Components for automatically syncing configured folders and detecting file changes (non-destructive import). Includes WebDAV logic for cloud-based synchronization.
- **`pkg/logger`:** Structured logging to trace application events and errors.
- **`pkg/worker`:** Background processing queues for long-running tasks, keeping the UI responsive.

## Data Flow

1. **User Action:** The user interacts with the Vue interface (e.g., uploading a tab).
2. **Wails Binding:** The frontend calls a bound Go method asynchronously over IPC.
3. **Business Logic:** The Go `app` package processes the request, validating inputs and determining necessary actions (e.g., calculating metadata, checking for duplicates).
4. **Data Persistence:** The `store` package inserts or updates records in the SQLite database. If new metadata needs fetching, it might queue a task to `worker` and `coverpool`.
5. **Response:** The Go method returns a result or an error back over the IPC bridge.
6. **State Update:** The frontend's Pinia store updates its local state and triggers a re-render of the Vue components.

## Deployment & Build Process

Wails bundles the frontend static assets directly into the Go binary. During `wails build`, Vite compiles the Vue application into standard HTML/JS/CSS, and Go compiles the backend along with the embedded frontend assets to produce a single native executable for the target platform (Windows, macOS, or Linux).
