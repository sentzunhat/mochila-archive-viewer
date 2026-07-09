# Snapchat Export Tool

Local tooling for inspecting and organizing Snapchat "My Data" exports.

## Local Intake

Put downloaded Snapchat export zips or extracted folders here:

```text
tools/snapchat-export/inbox/
```

This directory is gitignored. Do not commit Snapchat export archives, generated manifests, media, logs, tokens, cookies, or signed URLs.

## First Inspection Goal

The first tool should inspect a downloaded export without modifying it:

```bash
npx tsx inspect.ts --input tools/snapchat-export/inbox/mydata~1783534875834.zip
```

Expected output:

- detected zip/folder structure
- JSON file names
- top-level fields and rough record counts
- likely media URL fields
- likely timestamp, caption, location, and conversation fields
- privacy-safe summary report

## Display Direction

A small local app is a good fit after inspection. Recommended path:

1. Build a normalized `manifest.json` from the Snapchat export.
2. Create a local gallery app that reads the manifest and media from disk.
3. Keep private source data out of git and serve it only from the local machine.

Svelte is likely the lighter choice for a personal archive browser. React is fine if we want richer existing gallery/data-table components.
