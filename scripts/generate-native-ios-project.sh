#!/usr/bin/env bash
set -euo pipefail

readonly version="2.45.4"
readonly cache_root="${RUNNER_TEMP:-${TMPDIR:-/tmp}}/haya-tab-build-tools"
readonly tool_root="${cache_root}/xcodegen-${version}"
readonly binary="${tool_root}/.build/release/xcodegen"

if [[ ! -x "${binary}" ]]; then
  if [[ -e "${tool_root}" && ! -d "${tool_root}/.git" ]]; then
    echo >&2 "Refusing to replace unexpected XcodeGen cache path: ${tool_root}"
    exit 2
  fi
  if [[ ! -d "${tool_root}/.git" ]]; then
    mkdir -p "${cache_root}"
    git clone --branch "${version}" --depth 1 \
      https://github.com/yonaskolb/XcodeGen.git "${tool_root}"
  fi
  swift build --package-path "${tool_root}" -c release --product xcodegen
fi

"${binary}" generate --spec mobile/ios/project.yml
