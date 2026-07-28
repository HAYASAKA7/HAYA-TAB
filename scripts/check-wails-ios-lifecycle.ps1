$ErrorActionPreference = 'Stop'

$moduleJson = go list -m -json github.com/wailsapp/wails/v3 | ConvertFrom-Json
if (-not $moduleJson.Dir -or -not (Test-Path -LiteralPath $moduleJson.Dir -PathType Container)) {
    [Console]::Error.WriteLine('Unable to locate the selected Wails module.')
    exit 2
}

$nativeSource = Get-ChildItem -LiteralPath $moduleJson.Dir -Recurse -File |
    Where-Object {
        ($_.Extension -in '.m', '.mm', '.h' -or $_.Name -eq 'Info.plist') -and
        $_.Name -notmatch '_test\.'
    }
$hasSceneDelegate = Select-String -LiteralPath $nativeSource.FullName -Quiet -Pattern 'UIScene|SceneDelegate'
$hasSceneManifest = Select-String -LiteralPath $nativeSource.FullName -Quiet -Pattern 'UIApplicationSceneManifest'

if (-not ($hasSceneDelegate -and $hasSceneManifest)) {
    [Console]::Error.WriteLine('Pinned Wails lacks UIScene lifecycle support. iOS 27 packaging remains blocked.')
    exit 27
}

Write-Output 'Pinned Wails exposes both UIScene lifecycle APIs and the application-scene manifest.'
