# US Civics Practice

A local React + Go practice app for the USCIS **128 Civics Questions and Answers (2025 version)**. It provides study-guide order, a shuffled full set, or a unique randomized subset; server-authoritative scoring; multi-answer controls; review; and clear treatment of questions with changing/location-specific answers.

## Run in development

Prerequisites: Go 1.22+ and Node 20+.

1. In `frontend`, run `pnpm install`, then `pnpm dev`.
2. In a second terminal, run `cd backend` then `go run .`.
3. Open the Vite address (normally `http://localhost:5173`). API requests are proxied to Go on `127.0.0.1:8080`.

## Production build

Run `pnpm build` in `frontend`; it writes compiled assets to `backend/frontend/dist`. Then run `go build -o us-civics-practice.exe .` in `backend`. The executable serves UI and API together at `http://127.0.0.1:8080`.

## Validation and tests

- `go test ./...` from `backend` checks quiz selection, grading and validation paths.
- The Go server validates `data/questions-2025.json` at startup. It requires IDs 1-128, nonempty official answers, valid counts/metadata, and non-overlapping distractors.

## Question bank provenance

`data/questions-2025.json` was generated from USCIS M-1778 (09/25), downloaded from the official [2025 PDF](https://www.uscis.gov/sites/default/files/document/questions-and-answers/2025-Civics-Test-128-Questions-and-Answers.pdf). It contains all 128 records in official order and the listed accepted-answer text. Question IDs 23, 29, 30, 38, 39, 57, 61, and 62 are tagged as changing or location-dependent and excluded from automatic scoring until local current-answer overrides are maintained. Check [USCIS test updates](https://www.uscis.gov/citizenship/testupdates) before an interview.

The current distractors are deliberately explicit non-answers, rather than borrowed civics facts that might accidentally be valid. Replacing them with reviewed, question-specific distractors is an appropriate content-quality follow-up.
