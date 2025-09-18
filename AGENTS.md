# Repository Guidelines

## Project Structure & Module Organization
- `main.go` contains the multicast chat application entry point, CLI parsing, and networking logic.
- `README.md` introduces usage and runtime flags; reference it when updating user-facing behaviour.
- `DebuggingConnectivity.md` captures multicast troubleshooting steps—extend it when new edge cases appear.
- `.gocache/` is a local build cache; avoid committing it by keeping `.gitignore` up to date.

## Build, Test, and Development Commands
- `go build ./...` compiles the chat binary; use `GOCACHE="$(pwd)/.gocache"` when working in restricted environments.
- `go run .` launches the chat directly for interactive testing; add flags such as `-iface` or `-ttl` as needed.
- `gofmt -w <files>` formats Go sources; run it before committing changes.

## Coding Style & Naming Conventions
- Follow standard Go formatting (`gofmt`); tabs for indentation, CamelCase for exported symbols, and short, descriptive local names.
- Keep helper functions (`interfaceLocalAddr`, `setMulticastTTL`) focused and document tricky logic with succinct comments only when necessary.
- New flags or configuration should mirror existing naming: lowercase, dashed forms (e.g., `-ttl`).

## Testing Guidelines
- No automated test suite yet; validate changes with `go build` and by exercising `go run .` across multiple hosts when possible.
- If adding tests, place them alongside the code (`*_test.go`) and ensure they can run with `go test ./...` without external dependencies.
- Document manual test matrices or network setups in `DebuggingConnectivity.md` for future reference.

## Commit & Pull Request Guidelines
- Write commits in the imperative mood (`Add TTL flag`, `Document socat workflow`) and keep them scoped to a single concern.
- Each pull request should summarise behaviour changes, list verification steps (`go build`, `go run`), and mention any outstanding issues or follow-ups.
- Include screenshots or terminal transcripts when UI/UX output changes (e.g., new flags or prompts).

## Security & Configuration Tips
- Multicast traffic often depends on network policy: note any required router or firewall adjustments in `DebuggingConnectivity.md`.
- Avoid hard-coding network credentials or secrets; favor CLI flags or environment variables for future extensions.
