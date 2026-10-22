# US Civics Practice

## Download and Run

1. Open this repository's **Releases** page.
2. Download the latest `CivicsPractice-v*-windows-x64.zip` file.
3. Extract the ZIP to a folder you can find again.
4. Double-click `USCivicsPractice.exe`.
5. Study in the browser window that opens.

No installation, account, or internet connection is needed after download. The app only runs a local server on your computer (`127.0.0.1`) and never uploads your data. Keep the small console window open while studying; close it or press **Ctrl+C** to close the app.

### Optional: use another device on your home network

By default, the app is available only on the computer where it is running. To make it available on your private local network, open PowerShell in the app folder and run:

```powershell
.\USCivicsPractice.exe --lan
```

The console prints an address such as `http://192.168.1.25:8080`; enter that address in a browser on your phone or tablet. You can choose another port with `--lan --port 8081`.

When Windows asks about firewall access, allow it on **Private networks** only. Do not use router port forwarding or enable Public-network access: the app has no sign-in, and anyone allowed onto the network can access the shared study data while it is running. Avoid using the app from multiple devices at the same time, because they share one active profile and unfinished quiz.

Windows may show a SmartScreen notice because this free application is not code-signed. Read the notice and use only a download you trust; do not disable Windows security protections.

To update, download and extract a newer ZIP, then run its executable. Your progress is preserved separately at `%APPDATA%\USCivicsPractice\history.json`. Stage 4 migrates existing progress into a default profile automatically. Use **Settings → Save a backup** before moving computers or deleting local application data, and **Restore from backup** to bring it back as a separate profile.

## About this study tool

US Civics Practice is an independent study tool, not an official USCIS product. It includes the [USCIS 2025 128 Civics Questions and Answers](https://www.uscis.gov/sites/default/files/document/questions-and-answers/2025-Civics-Test-128-Questions-and-Answers.pdf). The real civics examination is oral. Some official answers can change over time or depend on where you live, so verify changing answers with [USCIS test updates](https://www.uscis.gov/citizenship/testupdates) before your interview.

The app supports a 20-question multiple-choice practice quiz, full 128-question sessions, study-order and shuffled practice, and self-graded oral practice. Incorrect multiple-choice responses show all officially acceptable answers.

## Current answers, profiles, and backups

**Current Answers** lists the eight civics questions whose answer can change: U.S. senators, U.S. representative, Speaker, President, Vice President, Chief Justice, governor, and state capital. Use the direct official links in that screen to verify an answer and save a manually confirmed response for the selected profile. Officeholder pages are intentionally not scraped automatically: an official page can change format or be ambiguous, and the app must not mistake an election winner for a current officeholder. State capitals use a small bundled reference.

Profiles keep their own location, settings, quiz history, unfinished practice, and current answers. Select or create one in Settings. Set your two-letter state and, if known, congressional district; the app will never infer a representative without a district.

Backups are regular JSON downloads containing the selected profile's progress and settings. Keep them private: they contain your study history. Restoring a backup always creates a new profile, so it does not overwrite existing progress.

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
