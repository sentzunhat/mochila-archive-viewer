# mochila-archive-viewer

Mochila is an open-source, local-first desktop archive viewer for exported social data — the source is public, but the app never sends your archive anywhere; everything stays on your machine. The live Go + Wails app sits under `src/`, starts with Snapchat, keeps raw exports on the local machine, and has modular provider lanes for Instagram and Facebook work that is still being staged.

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
- The app checks GitHub for a newer release once per launch (Settings → About → "Check for updates"), a single unauthenticated API call with a ~3s timeout. It fails silently if you're offline — the app is fully usable with no network access — and requires this repo to stay public.

## License

MPL-2.0 — see `LICENSE`.

## Notes

- The current Wails shell in `src/` already carries over the warm archive theme from the prototype.
- Snapchat is currently the only live supported provider in the app shell.
- Instagram and Facebook provider work exists in the repo, but the live shell still routes the user through Snapchat-only archive access.
- The old prototype stays in `archive/snapchat-export/` until the Go app reaches full feature parity across all providers.
- Local indexed metadata lives in `~/.mochila/database.sqlite` (SQLite).
- Provider-specific cached artifacts live under `~/.mochila/indexed/providers/<provider>/`.
