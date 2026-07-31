# README Visual Refresh Design

## Goal

Make the HAYA-TAB README persuade musicians to try the stable desktop
application. The page should explain the product quickly, reserve clear
positions for screenshots the maintainer will capture later, and route
developer detail into the existing documentation.

## Audience and Scope

The primary audience is guitarists and other musicians evaluating HAYA-TAB as
an application. The main README presents the stable Windows, macOS, and Linux
experience. Experimental mobile work remains in the mobile development
documentation and is not presented as an equally supported release platform.

## Chosen Approach

Use a product-story layout:

1. Establish the product promise and download path.
2. Show the populated library as the visual anchor.
3. Demonstrate the PDF and AlphaTab practice experiences.
4. Show WebDAV as the continuity story across a musician's computers.
5. Move implementation details into linked developer documentation.

This balances visual proof with page length and maintenance cost. A single
hero image would undersell the built-in viewers, while a large screenshot
gallery would become slow and difficult to keep current.

## README Structure

The optimized README uses this order:

1. App name and a concise product statement.
2. Desktop platform, release, and license badges.
3. A prominent release download link.
4. A reserved full-width library screenshot position.
5. Three musician-facing value points: organize, practice, and sync.
6. A `See HAYA-TAB in action` section with reserved positions for:
   - PDF annotation;
   - AlphaTab playback and looping;
   - the cloud library and WebDAV workflow.
7. A supported-formats table.
8. Condensed capabilities grouped under Organize, Practice, Sync, and
   Customize.
9. A five-step quick start.
10. Installation instructions, with the macOS unsigned-app guidance inside a
    collapsed details block.
11. Links to development, mobile development, architecture, contributing, and
    WebDAV documentation.
12. Contribution, license, and author information.

The external repository-activity image is removed because it does not help a
musician understand or install the application.

## Screenshot Placeholders

The maintainer will take all screenshots. Until those files exist, the README
contains hidden HTML comments at the intended insertion points rather than
broken image links or visible `coming soon` panels.

Each placeholder records the expected path and visual purpose:

- `docs/assets/readme/library.webp` — populated desktop library, 16:9;
- `docs/assets/readme/pdf-annotations.webp` — PDF viewer with the annotation
  controls and a visible annotation;
- `docs/assets/readme/alphatab-player.webp` — AlphaTab notation with playback
  and looping controls visible;
- `docs/assets/readme/cloud-library.webp` — cloud library or WebDAV file
  workflow without credentials or private server details.

The comments include ready-to-uncomment Markdown or HTML image markup,
descriptive alt text, and sizing guidance. Screenshots should share one theme,
window treatment, and sample library. They must not expose personal file paths,
credentials, server addresses, or other private data.

## Content Changes

The current long feature inventory becomes four short groups:

- **Organize:** supported formats, uploads and links, categories, tags, search,
  metadata, artwork, and batch operations.
- **Practice:** PDF reading and annotation, AlphaTab playback and looping,
  auto-scroll, and MIDI pedal controls.
- **Sync:** watched folders, WebDAV volumes, cloud access, and non-destructive
  behavior.
- **Customize:** themes, languages, key bindings, storage locations, and
  plugins.

Technical terms such as SQLite FTS5, worker-pool implementation details, and
fingerprint bucket internals remain in the architecture and WebDAV documents.
The README may mention the resulting user benefit without leading with the
implementation.

## Status Accuracy

The primary platform badge lists Windows, macOS, and Linux. Mobile development
remains accessible through the documentation links but is not advertised as a
stable end-user platform. Version information should use a release-backed badge
or link where practical to reduce manual drift.

## Validation

Before considering the refresh complete:

1. Preview the README in a GitHub-compatible Markdown renderer.
2. Check every local documentation and license link.
3. Confirm the page remains coherent while screenshot markup is commented out.
4. After screenshots are added, verify their alt text, rendered widths, file
   sizes, and readability in both wide and narrow README views.
5. Confirm screenshots contain no credentials, private paths, or personal
   library data.

