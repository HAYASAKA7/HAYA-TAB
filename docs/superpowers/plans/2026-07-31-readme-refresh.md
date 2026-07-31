# README Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the main README as a musician-first desktop product page with hidden, ready-to-use screenshot positions.

**Architecture:** Keep `README.md` concise and user-facing while routing technical detail into the existing `docs/` guides. Keep screenshot markup inside HTML comments until the maintainer adds the four WebP files, so the published README never contains broken images.

**Tech Stack:** GitHub Flavored Markdown, inline HTML, Shields.io, PowerShell validation, Go test suite

---

### Task 1: Rewrite the README product story

**Files:**
- Modify: `README.md:1`

- [ ] **Step 1: Replace `README.md` with the approved content**

Use the following complete document:

````markdown
# HAYA-TAB

Organize, read, play, and sync your music tabs from one lightweight desktop app.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue)
[![Latest release](https://img.shields.io/github/v/release/HAYASAKA7/HAYA-TAB?label=release)](https://github.com/HAYASAKA7/HAYA-TAB/releases/latest)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**[Download the latest release →](https://github.com/HAYASAKA7/HAYA-TAB/releases/latest)**

<!--
README SCREENSHOT: Library overview
Add docs/assets/readme/library.webp (recommended 1600x900, 16:9).
Capture a populated desktop library with album artwork, categories, search, and
the sidebar visible. Do not include personal file paths or private library data.
Then uncomment this block:
<p align="center">
  <img src="docs/assets/readme/library.webp" alt="HAYA-TAB desktop library showing tabs, categories, search, and album artwork" width="900">
</p>
-->

HAYA-TAB helps musicians:

- Keep PDF, Guitar Pro, and MusicXML files in one searchable library.
- Practice without leaving the app using built-in PDF and AlphaTab viewers.
- Carry the same organized library between computers through WebDAV.

## See HAYA-TAB in action

### Read, annotate, and perform

Open PDFs inside HAYA-TAB, use hands-free auto-scroll, and add pen or highlighter
notes on a non-destructive layer that leaves the original score untouched.

Open Guitar Pro and MusicXML files with AlphaTab to view notation, play parts,
loop sections, and adjust playback speed while you practice.

<!--
README SCREENSHOTS: Practice viewers
Add:
- docs/assets/readme/pdf-annotations.webp (recommended 1200x750)
- docs/assets/readme/alphatab-player.webp (recommended 1200x750)
Use the same theme and window treatment. Show a visible annotation in the PDF
capture and playback plus loop controls in the AlphaTab capture. Do not include
personal paths, private scores, or identifying library data.
Then uncomment this block:
<table>
  <tr>
    <td width="50%">
      <img src="docs/assets/readme/pdf-annotations.webp" alt="HAYA-TAB PDF viewer with non-destructive annotation controls" width="100%">
    </td>
    <td width="50%">
      <img src="docs/assets/readme/alphatab-player.webp" alt="HAYA-TAB AlphaTab viewer with playback and looping controls" width="100%">
    </td>
  </tr>
  <tr>
    <td align="center"><strong>PDF reading and annotations</strong></td>
    <td align="center"><strong>AlphaTab playback and looping</strong></td>
  </tr>
</table>
-->

### Keep your cloud library close

Connect a WebDAV server to discover cloud volumes, browse remote tabs, stream
files on demand, and download the pieces you want available offline. HAYA-TAB
preserves metadata and category organization across computers.

<!--
README SCREENSHOT: Cloud library
Add docs/assets/readme/cloud-library.webp (recommended 1600x900, 16:9).
Show the cloud library or remote file workflow with useful volume and file
context. Hide credentials, server addresses, account names, and private paths.
Then uncomment this block:
<p align="center">
  <img src="docs/assets/readme/cloud-library.webp" alt="HAYA-TAB cloud library browsing tabs stored on WebDAV volumes" width="900">
</p>
-->

## Supported formats

| Format | Extensions | Built-in experience |
| --- | --- | --- |
| PDF | `.pdf` | Reader, auto-scroll, and non-destructive annotations |
| Guitar Pro | `.gp`, `.gp5`, `.gpx` | Notation, playback, looping, section playback, and speed control |
| MusicXML | `.xml`, `.musicxml`, `.mxl` | Notation and playback through AlphaTab |

## What you can do

- **Organize** — Upload files or link them in place, group tabs into categories,
  add part/version tags, search titles, artists, and albums, and manage multiple
  tabs in a batch.
- **Practice** — Read and annotate PDFs, play Guitar Pro and MusicXML scores,
  loop sections, change speed, use auto-scroll, and map viewer actions to a MIDI
  foot controller.
- **Sync** — Watch local folders for new files and use WebDAV volumes for
  multi-computer access, on-demand cloud files, uploads, and lightweight
  annotation sync.
- **Customize** — Choose light or dark themes, configure key bindings and
  storage locations, use English, Simplified or Traditional Chinese, or
  Japanese, and extend metadata workflows with JavaScript plugins.

## Quick start

1. **Add music:** Right-click empty library space and choose **Upload TAB** or
   **Link Local TAB**.
2. **Organize it:** Create categories, edit metadata, and add part or version
   tags.
3. **Bring in folders:** Add sync paths in Settings to import supported files
   automatically.
4. **Connect your cloud:** Configure WebDAV in Settings to browse and sync a
   remote library. See the [WebDAV guide](docs/WEBDAV.md).
5. **Start practicing:** Open a tab normally or choose **Open with Inner
   Viewer** for PDF and AlphaTab practice tools.

## Installation

Download the current build for your operating system from
[GitHub Releases](https://github.com/HAYASAKA7/HAYA-TAB/releases/latest).

<details>
<summary>macOS reports that HAYA-TAB cannot be verified or is damaged</summary>

HAYA-TAB is not currently signed with an Apple Developer certificate.

1. Open **System Settings → Privacy & Security**.
2. Find the blocked HAYA-TAB launch and select **Open Anyway**.
3. If macOS still blocks the app after you move it into `/Applications`, run:

   ```bash
   xattr -cr /Applications/HAYA-TAB.app
   ```

</details>

## Documentation

- [Development guide](docs/DEVELOPMENT.md)
- [Mobile development guide](docs/MOBILE_DEVELOPMENT.md) — experimental
- [Architecture overview](docs/ARCHITECTURE.md)
- [Contributing guidelines](docs/CONTRIBUTING.md)
- [WebDAV guide](docs/WEBDAV.md)

## License

HAYA-TAB is available under the [Apache License 2.0](LICENSE). See
[NOTICE](NOTICE) for attribution information.

## Author

**HAYASAKA7** — [cyanluxury267@gmail.com](mailto:cyanluxury267@gmail.com)
````

- [ ] **Step 2: Inspect the user-facing diff**

Run `git diff -- README.md`.

Expected: the long flat feature list and repository-activity image are gone;
the stable platform badge contains Windows, macOS, and Linux; the download link
is prominent; all four screenshot paths occur only inside HTML comments.

- [ ] **Step 3: Commit the README rewrite**

Run:

```powershell
git add README.md
git commit -m "docs: refresh README product story"
```

Expected: one commit containing only `README.md`.

### Task 2: Validate the README contract

**Files:**
- Verify: `README.md`

- [ ] **Step 1: Verify required sections and screenshot positions**

Run:

```powershell
$readme = Get-Content -Raw README.md
$required = @(
  '## See HAYA-TAB in action',
  '## Supported formats',
  '## What you can do',
  '## Quick start',
  '## Installation',
  'docs/assets/readme/library.webp',
  'docs/assets/readme/pdf-annotations.webp',
  'docs/assets/readme/alphatab-player.webp',
  'docs/assets/readme/cloud-library.webp'
)
$missing = $required | Where-Object { -not $readme.Contains($_) }
if ($missing) { throw "Missing README content: $($missing -join ', ')" }
if ($readme.Contains('repopulse')) { throw 'Repository activity image remains' }
if ($readme -match 'platform-[^\r\n)]*iOS') { throw 'Platform badge advertises iOS' }
'README structure OK'
```

Expected: `README structure OK`.

- [ ] **Step 2: Verify local Markdown links**

Run:

```powershell
$readme = Get-Content -Raw README.md
$targets = [regex]::Matches($readme, '\[[^\]]+\]\((?!https?://|mailto:|#)([^)]+)\)') |
  ForEach-Object { $_.Groups[1].Value } |
  Sort-Object -Unique
$missing = $targets | Where-Object { -not (Test-Path -LiteralPath $_) }
if ($missing) { throw "Broken local README links: $($missing -join ', ')" }
'README links OK'
```

Expected: `README links OK`.

- [ ] **Step 3: Run the repository tests**

Run `go test ./...`.

Expected: all packages pass.

- [ ] **Step 4: Confirm final scope**

Run:

```powershell
git status --short
git show --stat --oneline HEAD
```

Expected: no uncommitted changes from the README implementation and the latest
implementation commit contains only `README.md`.

