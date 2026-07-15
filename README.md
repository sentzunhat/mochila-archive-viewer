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

## Notes

- The current Wails shell in `src/` already carries over the warm archive theme from the prototype.
- Snapchat is currently the only live supported provider in the app shell.
- Instagram and Facebook provider work exists in the repo, but the live shell still routes the user through Snapchat-only archive access.
- The old prototype stays in `archive/snapchat-export/` until the Go app reaches full feature parity across all providers.
- Local indexed metadata lives in `~/.mochila/database.sqlite` (SQLite).
- Provider-specific cached artifacts live under `~/.mochila/indexed/providers/<provider>/`.
