# US Civics Practice

US Civics Practice is a Windows study aid for the **2025 USCIS 128-question civics test**. It lets you practice all 128 questions in study or shuffled order, take a smaller practice set, and use an oral self-grading mode that better resembles the real interview. It is an independent project and **not** an official USCIS product.

## Download and run on Windows

1. Open the repository's [Releases](https://github.com/rmclimber/us_civics_test/releases) page.
2. Under the release you want, download `CivicsPractice-v*-windows-x64.zip`.
3. In File Explorer, right-click the downloaded ZIP and choose **Extract All**.
4. Open the extracted folder and double-click `USCivicsPractice.exe`.
5. Study in the browser window that opens.

No developer tools or account are required. The app runs locally on your computer. It normally needs no internet connection after it has been downloaded, but opening official-source links and any optional online functionality requires an internet connection.

Keep the small console window open while studying. To close the application, close that console window or press **Ctrl+C** in it; then close the browser tab if it remains open.

Windows may show a SmartScreen notice because the executable is not assumed to be code-signed. Read the notice, verify that the download came from this project's Releases page, and follow your organization's or Windows security guidance. Do not disable security protections.

### Optional: use another device on your home network

By default, the app is available only on the computer where it is running. To make it available on a private local network, open PowerShell in the extracted app folder and run:

```powershell
.\USCivicsPractice.exe --lan
```

The console prints an address such as `http://192.168.1.25:8080`; enter it in a browser on your phone or tablet. You can choose a port with `--lan --port 8081`. If Windows asks about firewall access, allow it on **Private networks** only. Do not set up router port forwarding or enable Public-network access. The app has no sign-in; devices allowed onto the private network can access its shared study data while it is running. Avoid simultaneous use from multiple devices.

## Using the app

The app offers 20-question multiple-choice practice by default, full 128-question sessions, study-order or shuffled questions, and oral, self-graded practice. Incorrect multiple-choice responses show the accepted answers.

### Changing answers

Some civics answers can change, including officeholders and your state capital. Open **Current** in the app, follow the direct official links, and manually confirm the answers before they are included in scored practice. The app does not scrape official pages or independently verify your entries. Review these answers before your interview and consult [USCIS test updates](https://www.uscis.gov/citizenship/testupdates).

### Profiles, history, and backups

Profiles separately store their settings, selected location, current answers, unfinished practice, and completed-study history on this computer. Create or switch profiles in **Settings**. The app will not infer a congressional representative without a district.

In **Settings → Backup and restore**, choose **Save a backup** to download a JSON copy of the selected profile. To restore one, choose **Restore from backup** and select the JSON file. Restoration creates a new profile and does not overwrite existing ones. Keep backup files private and copy them somewhere safe before moving computers: local data can be lost through device failure, deletion, or reset and should be backed up.

## Important information and disclaimers

This application is an independent study aid, not an official USCIS product. It includes the [USCIS 2025 128 Civics Questions and Answers](https://www.uscis.gov/sites/default/files/document/questions-and-answers/2025-Civics-Test-128-Questions-and-Answers.pdf) as source material; USCIS-authored materials remain subject to their own terms, and third-party software remains subject to its own licenses. Nothing here claims exclusive ownership of those materials.

No guarantee is made that the content is accurate, complete, current, or that using the app will lead to examination success. Dynamic answers can become outdated. Consult the linked official government sources, including [USCIS test updates](https://www.uscis.gov/citizenship/testupdates), before relying on an answer for an immigration interview.

To the extent permitted by applicable law, the application is provided **as-is**, without warranties of accuracy, reliability, security, or fitness for a particular purpose. It has not been independently security-audited. User study history and profile data are stored locally; they are not guaranteed against loss and should be backed up. No unsupported privacy or security guarantees are made.

## License and third-party notices

This project is **source-available, not OSI-approved open source**. It is licensed under the [PolyForm Noncommercial License 1.0.0](LICENSE). In plain language, you may use, modify, and share it for noncommercial purposes; commercial use requires separate permission from the copyright holder.

Copyright 2026 Rick Morris. For commercial permission, open a [GitHub issue](https://github.com/rmclimber/us_civics_test/issues) or contact the project owner at `rmorris.code@gmail.com`.

The distributed application includes [third-party notices](THIRD_PARTY_NOTICES.md) for React, React DOM, and Lucide React. Those components use permissive third-party licenses and do not change the noncommercial license for the application's original code.

## Problems and support

Report bugs or release-package problems through the [GitHub issue tracker](https://github.com/rmclimber/us_civics_test/issues). For question or answer corrections, include the question number and an official source link where possible.

## Developer and release notes

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

Run backend validation and tests:

```powershell
cd backend
go test ./...
```

Build a Windows x64 release; the frontend assets and question bank are compiled into the executable:

```powershell
.\scripts\build-release.ps1 -Version 0.9.0
```

The output is `release\CivicsPractice-v0.9.0-windows-x64.zip`. It contains the executable, user instructions, this license, and third-party notices. The GitHub Actions workflow runs the frontend build, question-bank validation, and backend tests before packaging a tagged prerelease.
