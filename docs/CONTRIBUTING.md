# Contributing to HAYA-TAB

Thank you for your interest in contributing to HAYA-TAB! Contributions of all forms—bug reports, feature requests, documentation improvements, and code—are welcome.

## Code of Conduct

Please be respectful and constructive when interacting with other community members in issues and pull requests.

## Development Environment Setup

### Prerequisites

To build and run HAYA-TAB locally, you need the following tools installed on your system:

- [Go](https://go.dev/dl/) (version 1.21 or later is recommended)
- [Node.js](https://nodejs.org/) (npm)
- [Wails](https://v3.wails.io/getting-started/installation/) (v3)

### Getting Started

1. **Fork and Clone the Repository:**
   ```bash
   git clone https://github.com/HAYASAKA7/HAYA-TAB.git
   cd HAYA-TAB
   ```

2. **Install Frontend Dependencies:**
   ```bash
   cd frontend
   npm install
   cd ..
   ```

3. **Run the Application in Development Mode:**
   Using Wails v3 dev mode provides hot-reloading for the frontend:
   ```bash
   wails3 task dev
   ```

## Workflow

### Submitting an Issue

If you encounter a bug or have a feature idea, please open an issue describing the problem or the proposed feature. Include as much detail as possible, such as OS version, HAYA-TAB version, and steps to reproduce any bugs.

### Submitting a Pull Request

1. Create a new branch for your feature or bug fix:
   ```bash
   git checkout -b feature/my-awesome-feature
   ```
2. Make your changes in the codebase.
3. Verify your changes do not break existing functionality. Run tests if applicable (`go test ./...` for Go backend).
4. If you modify the database schema, ensure the migration logic (`pkg/store/migration.go`) is updated appropriately.
5. If you modify or add backend APIs, ensure `docs/API.md` is updated or regenerate it using `go doc`.
6. Commit your changes with clear, descriptive commit messages.
7. Push the branch to your fork and open a Pull Request against the main repository.

## Coding Guidelines

- **Go (Backend):** Adhere to standard Go formatting (`gofmt`). Ensure robust error handling and avoid panic. Follow idiomatic Go naming conventions.
- **Vue (Frontend):** Use Vue 3 Composition API with `<script setup>`. Use TypeScript for strong typing. Manage global state with Pinia (`src/stores`).
- **Documentation:** Keep documentation (like `README.md`, `ARCHITECTURE.md`, and inline comments) up-to-date with your changes.

## Plugin Development

HAYA-TAB plugins are JavaScript modules loaded by `PluginManager` at startup from the user app directory:

- `<os.UserConfigDir()>/HAYA-TAB/plugins/<plugin-id>/`

For this repository, built-in/distributed plugins are maintained in:

- `internal/app/plugins/<plugin-id>/`

### Plugin Coding Guidelines

- Keep plugin files self-contained and deterministic. A plugin should either return useful data or return `null`/original data without side effects.
- Fail gracefully: wrap parsing/network logic with guards and return safely when inputs are invalid or API calls fail.
- Never hardcode secrets in `index.js` or `manifest.json`. Use `config` values from `config.json`.
- Validate and sanitize all external data (HTTP responses, JSON payloads) before applying it to metadata fields.
- Keep logs concise and prefixed (for example `[my-plugin] ...`) so plugin behavior is easy to debug.
- Keep compatibility with the current runtime (`goja` JS execution); use plain JavaScript patterns already used in existing plugins.
- Only request permissions/hooks your plugin needs in `manifest.json`.

### How to Add a Plugin

1. Create a new plugin directory:
   - `internal/app/plugins/<plugin-id>/`
2. Add `manifest.json` with:
   - `id`, `name`, `version`, `entry`, `hooks`, `permissions`, and optional `settingsSchema`.
3. Add the entry file referenced by `entry` (usually `index.js`).
4. Export hook functions that match declared hooks:
   - `metadata` hook requires `module.exports.enhanceMetadata = function(tab) { ... return tab; }`
   - `cover` hook requires `module.exports.getCoverUrl = function(artist, album, title, country, lang) { ... return urlOrNull; }`
5. (Recommended) Add `config.json.example` documenting required settings.
6. Test by copying/syncing the plugin folder into local runtime plugins directory:
   - `<os.UserConfigDir()>/HAYA-TAB/plugins/<plugin-id>/`
7. Start the app (`wails3 task dev`) and verify:
   - The plugin appears in settings.
   - Hook behavior works without runtime errors.
8. After `git push`, sync plugin subtree to the plugins repository:
   - One-time setup (if `plugins-repo` remote does not exist yet):
   ```bash
   git remote add plugins-repo https://github.com/HAYASAKA7/HAYA-TAB-Plugins.git
   ```
   - Push the plugins subtree:
   ```bash
   git subtree push --prefix=internal/app/plugins plugins-repo main
   ```

### Recommended Plugin Structure

```text
<plugin-id>/
  manifest.json
  index.js
  config.json.example   # optional but recommended
  README.md             # optional plugin-specific docs
```

`manifest.json` template:

```json
{
  "id": "my-plugin",
  "name": "My Plugin",
  "version": "1.0.0",
  "entry": "index.js",
  "hooks": ["metadata"],
  "permissions": ["network"],
  "settingsSchema": {
    "apiKey": "password",
    "baseUrl": "string"
  }
}
```

Runtime globals available in plugin scripts:

- `log(message)` for plugin logs.
- `fetch(url)` for simple GET requests.
- `httpRequest({ method, url, headers, body })` for advanced HTTP.
- `config` object loaded from `config.json` (if present).

## Building for Production

To create a production-ready binary for your current platform:

```bash
wails3 task build
```

The resulting executable will be located in the `bin/` directory.
If cross-compiling from a different host OS, run `wails3 task setup:docker` once first.

---
Thank you for helping improve HAYA-TAB!
