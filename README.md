# US Civics Practice

## Download and Run

1. Open this repository's **Releases** page.
2. Download the latest `CivicsPractice-v*-windows-x64.zip` file.
3. Extract the ZIP to a folder you can find again.
4. Double-click `USCivicsPractice.exe`.
5. Study in the browser window that opens.

No installation, account, or internet connection is needed after download. The app only runs a local server on your computer (`127.0.0.1`) and never uploads your data. Keep the small console window open while studying; close it or press **Ctrl+C** to close the app.

Windows may show a SmartScreen notice because this free application is not code-signed. Read the notice and use only a download you trust; do not disable Windows security protections.

To update, download and extract a newer ZIP, then run its executable. Your progress is preserved separately at `%APPDATA%\USCivicsPractice\history.json`. To back it up, close the app and copy that file somewhere safe. The file contains completed attempts, an unfinished session, theme, and practice preferences.

## About this study tool

US Civics Practice is an independent study tool, not an official USCIS product. It includes the [USCIS 2025 128 Civics Questions and Answers](https://www.uscis.gov/sites/default/files/document/questions-and-answers/2025-Civics-Test-128-Questions-and-Answers.pdf). The real civics examination is oral. Some official answers can change over time or depend on where you live, so verify changing answers with [USCIS test updates](https://www.uscis.gov/citizenship/testupdates) before your interview.

The app supports a 20-question multiple-choice practice quiz, full 128-question sessions, study-order and shuffled practice, and self-graded oral practice. Incorrect multiple-choice responses show all officially acceptable answers.

## Developer notes

Prerequisites: Go 1.22+, Node 20+, and Corepack/pnpm.

For local development:

```powershell
cd frontend
corepack pnpm install
corepack pnpm dev
# in a second PowerShell window
cd backend
go run .
```

The Vite development server proxies API requests to Go at `127.0.0.1:8080`.

Validate and test the backend:

```powershell
cd backend
go test ./...
```

Build a reproducible Windows x64 release (frontend assets and question bank are compiled into the executable):

```powershell
.\scripts\build-release.ps1 -Version 1.0.0
```

The output is `release\CivicsPractice-v1.0.0-windows-x64.zip`. The GitHub Actions workflow runs these checks, packages the ZIP, and publishes it to GitHub Releases whenever a `v*` tag is pushed.

## Privacy and data safety

The application has no analytics, telemetry, or ordinary-use network calls. It binds only to the local loopback address and does not enable cross-origin API access. Local data is schema-versioned and saved atomically; an incompatible or corrupt history file is left unchanged and reported instead of being silently reset.
