# HAYA-TAB-Plugins

This repository serves as the official plugin distribution for **HAYA-TAB**. It contains a collection of extensions designed to enhance the functionality and user experience of the HAYA-TAB platform.

## Usage

At runtime, HAYA-TAB loads plugins from the user config directory:

- `<os.UserConfigDir()>/HAYA-TAB/plugins/<plugin-id>/`

To install a plugin, copy the plugin folder into that directory.

## Plugin Structure

Each plugin should follow this structure:

```text
<plugin-id>/
  manifest.json
  index.js
  config.json.example   # optional
```

### `manifest.json` fields

- `id`: unique plugin ID (should match folder name)
- `name`: display name
- `version`: semantic version
- `entry`: entry script path (usually `index.js`)
- `hooks`: supported hooks (`metadata`, `cover`)
- `permissions`: required permissions (for example `network`)
- `settingsSchema`: optional settings definition for UI

### Required exports by hook

- `metadata` hook -> `module.exports.enhanceMetadata = function(tab) { return tab; }`
- `cover` hook -> `module.exports.getCoverUrl = function(artist, album, title, country, lang) { return null; }`

## Note

This repository is synchronized automatically. If you wish to contribute or report issues, please refer to the main HAYA-TAB project repository.
