param(
    [string]$Version = "1.0.0"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$frontend = Join-Path $root "frontend"
$backend = Join-Path $root "backend"
$release = Join-Path $root "release"
New-Item -ItemType Directory -Path $release -Force | Out-Null

Push-Location $frontend
try {
    corepack pnpm install --frozen-lockfile
    corepack pnpm build
} finally { Pop-Location }

Push-Location $backend
try {
    go test ./...
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go build -trimpath -ldflags="-s -w" -o (Join-Path $release "USCivicsPractice.exe") .
} finally {
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location
}

Copy-Item (Join-Path $root "README.md") (Join-Path $release "README.txt") -Force
Copy-Item (Join-Path $root "LICENSE") (Join-Path $release "LICENSE.txt") -Force
Copy-Item (Join-Path $root "THIRD_PARTY_NOTICES.md") (Join-Path $release "THIRD_PARTY_NOTICES.txt") -Force
Compress-Archive -Path (Join-Path $release "USCivicsPractice.exe"), (Join-Path $release "README.txt"), (Join-Path $release "LICENSE.txt"), (Join-Path $release "THIRD_PARTY_NOTICES.txt") -DestinationPath (Join-Path $release "CivicsPractice-v$Version-windows-x64.zip") -Force
Write-Host "Release package created in $release"
