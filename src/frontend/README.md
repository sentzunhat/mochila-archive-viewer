# Mochila Frontend

This is the Svelte + Vite frontend used by the live Wails desktop app in `src/`.

## Responsibilities

- Render the warm local-first Mochila desktop UI
- Show gallery, On This Day, messages, structure, and JSON data explorer views
- Call Wails-generated bindings from `../wailsjs/` for archive indexing and local data access

## Development

```bash
cd src/frontend
npm install
npm run dev
```

Production build check:

```bash
npm run build
```

## Storage expectations

- SQLite metadata DB: `~/.mochila/database.sqlite`
- Provider cache root: `~/.mochila/indexed/providers/`
- Snapchat media cache: `~/.mochila/indexed/providers/snapchat/media/`
- Snapchat snapshot file: `~/.mochila/indexed/providers/snapchat/snapshot.json`

The UI should describe the same paths shown by the backend structure view.
