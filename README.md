# mochila-archive-viewer

Mochila is a private local-first desktop archive viewer for exported social data. The live Go + Wails app now sits under `src/`, starts with Snapchat, keeps raw exports on the local machine, and has modular provider lanes for Instagram and Facebook work that is still being staged.

## Layout

- `src/`: live Wails project root
- `src/main.go`: Wails entrypoint (Go module path: `mochila-archive-viewer/src`)
- `src/appshell/`, `src/internal/`: Go app source and provider/archive services
- `src/frontend/`: desktop UI (Svelte + Vite)
- `archive/snapchat-export/`: archived Svelte + Node proof of concept
- `archive/docs/`: earlier planning and inspection notes

## Local development

```bash
cd src && wails dev
```

Fast verification:

```bash
cd src && go test ./...
cd src/frontend && npm run build
```

## Releases

Releases are built by CI (`.github/workflows/release.yml`) when a version tag is pushed:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow builds macOS (Apple Silicon + Intel) and Windows binaries with the tag embedded as the app version (visible in Settings → About), and attaches them to a **draft** GitHub Release — review and publish it from the Releases page.

- App identity lives in `src/wails.json` (`info` block) and `src/build/darwin/Info.plist` (bundle id `com.sentzunhat.mochila-archive-viewer`); the app icon is `src/build/appicon.png`.
- Binaries are currently unsigned: on macOS, downloaded builds need right-click → Open the first time (Gatekeeper).
- An in-app update check against GitHub Releases is planned — see `.hawp/work/active/013.md`.

## Notes

- The current Wails shell in `src/` already carries over the warm archive theme from the prototype.
- Snapchat is currently the only live supported provider in the app shell.
- Instagram and Facebook provider work exists in the repo, but the live shell still routes the user through Snapchat-only archive access.
- The old prototype stays in `archive/snapchat-export/` until the Go app reaches full feature parity across all providers.
- Local indexed metadata lives in `~/.mochila/database.sqlite` (SQLite).
- Provider-specific cached artifacts live under `~/.mochila/indexed/providers/<provider>/`.
