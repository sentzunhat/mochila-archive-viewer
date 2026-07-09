# mochila-archive-viewer

Mochila is a private local-first desktop archive viewer for exported social data. The live Go + Wails app now sits under `src/`, starts with Snapchat, keeps raw exports on the local machine, and prepares modular provider lanes for Instagram and Facebook.

## Layout

- `src/`: live Wails project root
- `src/main.go`: Wails entrypoint kept at the project root for the CLI
- `src/appshell/`, `src/internal/`: Go app source and provider/archive services
- `src/frontend/`: desktop UI
- `archive/snapchat-export/`: archived Svelte + Node proof of concept
- `archive/docs/`: earlier planning and inspection notes
- `inbox/`: local private exports only, kept out of git

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
- Snapchat is the only active provider target right now.
- The old prototype stays in `archive/snapchat-export/` until the Go app reaches feature parity.
- Local indexed metadata lives in `~/.mochila/database.sqlite`.
- Provider-specific cached artifacts live under `~/.mochila/indexed/providers/<provider>/`.
- For Snapchat, cached media is stored in `~/.mochila/indexed/providers/snapchat/media/` and the snapshot file in `~/.mochila/indexed/providers/snapchat/snapshot.json`.
