#!/usr/bin/env bash
set -euo pipefail

module_dir="$(go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v3)"
if [[ -z "$module_dir" || ! -d "$module_dir" ]]; then
  echo >&2 "Unable to locate the selected Wails module."
  exit 2
fi

native_files=(--glob '*.m' --glob '*.mm' --glob '*.h' --glob 'Info.plist' --glob '!**/*_test.*')

if ! rg -q "UIScene|SceneDelegate" "${native_files[@]}" "$module_dir" ||
   ! rg -q "UIApplicationSceneManifest" "${native_files[@]}" "$module_dir"; then
  echo >&2 "Pinned Wails lacks UIScene lifecycle support. iOS 27 packaging remains blocked."
  exit 27
fi

echo "Pinned Wails exposes both UIScene lifecycle APIs and the application-scene manifest."
