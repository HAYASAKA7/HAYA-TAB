# Contributing to HAYA-TAB

Thank you for your interest in contributing to HAYA-TAB! Contributions of all forms—bug reports, feature requests, documentation improvements, and code—are welcome.

## Code of Conduct

Please be respectful and constructive when interacting with other community members in issues and pull requests.

## Development Environment Setup

### Prerequisites

To build and run HAYA-TAB locally, you need the following tools installed on your system:

- [Go](https://go.dev/dl/) (version 1.21 or later is recommended)
- [Node.js](https://nodejs.org/) (npm)
- [Wails](https://wails.io/docs/gettingstarted/installation) (v2)

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
   Using Wails dev mode provides hot-reloading for the frontend:
   ```bash
   wails dev
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

## Building for Production

To create a production-ready binary for your current platform:

```bash
wails build
```

The resulting executable will be located in the `build/bin/` directory.

---
Thank you for helping improve HAYA-TAB!
