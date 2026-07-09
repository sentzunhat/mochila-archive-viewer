#!/usr/bin/env node

import { createServer } from "node:http";
import { spawn, spawnSync } from "node:child_process";
import { extname, join, resolve } from "node:path";
import { statSync } from "node:fs";
import { fileURLToPath } from "node:url";

const appDir = resolve(fileURLToPath(new URL(".", import.meta.url)));
const toolDir = resolve(appDir, "..");
const inboxDir = join(toolDir, "inbox");
const port = Number(process.env.PORT || 4177);
const zipPattern = /^mydata~.*\.zip$/;
const mediaPattern = /\.(jpe?g|png|webp|gif|heic|mp4|mov|webm)$/i;
const imagePattern = /\.(jpe?g|png|webp|gif)$/i;
const videoPattern = /\.(mp4|mov|webm)$/i;

function fail(message) {
  console.error(message);
  process.exit(1);
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, { encoding: "utf8", ...options });
  if (result.status !== 0) {
    throw new Error(result.stderr || `${command} exited with ${result.status}`);
  }
  return result.stdout;
}

function listZipFiles() {
  const output = run("find", [inboxDir, "-maxdepth", "1", "-type", "f", "-name", "mydata~*.zip", "-print"]);
  return output
    .split("\n")
    .filter(Boolean)
    .map((path) => resolve(path))
    .filter((path) => zipPattern.test(path.split("/").pop() || ""))
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true }));
}

function listEntries(zipPath) {
  return run("unzip", ["-Z1", zipPath])
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
}

function mimeFor(entry) {
  const ext = extname(entry).toLowerCase();
  if (ext === ".jpg" || ext === ".jpeg") return "image/jpeg";
  if (ext === ".png") return "image/png";
  if (ext === ".webp") return "image/webp";
  if (ext === ".gif") return "image/gif";
  if (ext === ".mp4") return "video/mp4";
  if (ext === ".mov") return "video/quicktime";
  if (ext === ".webm") return "video/webm";
  if (ext === ".json") return "application/json";
  if (ext === ".html") return "text/html; charset=utf-8";
  return "application/octet-stream";
}

function dateFromEntry(entry) {
  const match = entry.match(/(?:^|\/)(\d{4}-\d{2}-\d{2})[_/]/);
  return match?.[1] || "unknown";
}

function typeFor(entry) {
  if (imagePattern.test(entry)) return "image";
  if (videoPattern.test(entry)) return "video";
  return "other";
}

function readZipText(zipPath, entry) {
  return run("unzip", ["-p", zipPath, entry], { maxBuffer: 128 * 1024 * 1024 });
}

function findJsonEntry(entryName) {
  return jsonFiles.find((file) => file.entry === entryName);
}

function readJsonEntry(entryName, fallback = null) {
  const item = findJsonEntry(entryName);
  if (!item) return fallback;

  const zip = zipMeta[item.zipIndex];
  return JSON.parse(readZipText(zip.path, item.entry));
}

function summarizeJson(value) {
  if (Array.isArray(value)) {
    return {
      kind: "array",
      records: value.length,
      sample: value.slice(0, 20),
      keys: value[0] && typeof value[0] === "object" ? Object.keys(value[0]).slice(0, 40) : [],
    };
  }

  if (value && typeof value === "object") {
    const object = value;
    const keys = Object.keys(object);
    const childCounts = keys
      .map((key) => {
        const child = object[key];
        return {
          key,
          type: Array.isArray(child) ? "array" : typeof child,
          records: Array.isArray(child) ? child.length : undefined,
        };
      })
      .slice(0, 80);

    return {
      kind: "object",
      records: keys.length,
      keys: keys.slice(0, 80),
      childCounts,
      sample: object,
    };
  }

  return { kind: typeof value, records: 1, keys: [], sample: value };
}

const zips = listZipFiles();
if (zips.length === 0) {
  fail(`No mydata~*.zip files found in ${toolDir}`);
}

console.log(`Indexing ${zips.length} Snapchat export zip files...`);

const zipMeta = zips.map((path, zipIndex) => {
  const name = path.split("/").pop();
  const entries = listEntries(path);
  const stat = statSync(path);
  return { zipIndex, path, name, entries, size: stat.size };
});

const media = [];
const jsonFiles = [];
const htmlFiles = [];
const yearCounts = new Map();
const categoryCounts = new Map();
const typeCounts = new Map();

for (const zip of zipMeta) {
  for (const entry of zip.entries) {
    const category = entry.split("/")[0] || "root";
    categoryCounts.set(category, (categoryCounts.get(category) || 0) + 1);

    if (entry.toLowerCase().endsWith(".json")) {
      jsonFiles.push({ zipIndex: zip.zipIndex, zip: zip.name, entry });
    }

    if (entry.toLowerCase().endsWith(".html")) {
      htmlFiles.push({ zipIndex: zip.zipIndex, zip: zip.name, entry });
    }

    if (!mediaPattern.test(entry)) continue;

    const date = dateFromEntry(entry);
    const year = date === "unknown" ? "unknown" : date.slice(0, 4);
    const type = typeFor(entry);
    yearCounts.set(year, (yearCounts.get(year) || 0) + 1);
    typeCounts.set(type, (typeCounts.get(type) || 0) + 1);

    media.push({
      id: media.length,
      zipIndex: zip.zipIndex,
      zip: zip.name,
      entry,
      category,
      date,
      year,
      type,
      ext: extname(entry).toLowerCase().replace(".", ""),
    });
  }
}

const summary = {
  generatedAt: new Date().toISOString(),
  zipCount: zipMeta.length,
  totalZipBytes: zipMeta.reduce((sum, zip) => sum + zip.size, 0),
  totalEntries: zipMeta.reduce((sum, zip) => sum + zip.entries.length, 0),
  mediaCount: media.length,
  jsonCount: jsonFiles.length,
  htmlCount: htmlFiles.length,
  zips: zipMeta.map((zip) => ({
    zipIndex: zip.zipIndex,
    name: zip.name,
    size: zip.size,
    entries: zip.entries.length,
  })),
  categories: Object.fromEntries([...categoryCounts.entries()].sort()),
  years: Object.fromEntries([...yearCounts.entries()].sort()),
  types: Object.fromEntries([...typeCounts.entries()].sort()),
  jsonFiles,
  htmlFiles: htmlFiles.slice(0, 300),
};

function normalizeMessage(message) {
  return {
    from: String(message.From ?? ""),
    content: String(message.Content ?? ""),
    mediaType: String(message["Media Type"] ?? ""),
    created: String(message.Created ?? ""),
    isSender: Boolean(message.IsSender),
    isSaved: Boolean(message.IsSaved),
    mediaIds: String(message["Media IDs"] ?? ""),
  };
}

function buildConversations() {
  const chatHistory = readJsonEntry("json/chat_history.json", {});
  if (!chatHistory || typeof chatHistory !== "object" || Array.isArray(chatHistory)) {
    return [];
  }

  return Object.entries(chatHistory)
    .map(([id, rawMessages]) => {
      const messages = Array.isArray(rawMessages) ? rawMessages.map(normalizeMessage) : [];
      const title = messages.find((message) => message.content || message.from)?.from || id;
      const lastCreated = messages
        .map((message) => message.created)
        .filter(Boolean)
        .sort()
        .at(-1) || null;

      return {
        id,
        title,
        messageCount: messages.length,
        savedCount: messages.filter((message) => message.isSaved).length,
        mediaCount: messages.filter((message) => message.mediaType || message.mediaIds).length,
        lastCreated,
        messages,
      };
    })
    .sort((a, b) => String(b.lastCreated || "").localeCompare(String(a.lastCreated || "")));
}

const conversations = buildConversations();

function sendJson(res, body) {
  const text = JSON.stringify(body);
  res.writeHead(200, {
    "content-type": "application/json; charset=utf-8",
    "cache-control": "no-store",
  });
  res.end(text);
}

function sendText(res, status, body, contentType = "text/plain; charset=utf-8") {
  res.writeHead(status, { "content-type": contentType });
  res.end(body);
}

function notFound(res) {
  sendText(res, 404, "Not found");
}

function parseUrl(req) {
  return new URL(req.url || "/", `http://${req.headers.host || "localhost"}`);
}

function streamZipEntry(res, item, download = false) {
  const zip = zipMeta[item.zipIndex];
  if (!zip) return notFound(res);

  const child = spawn("unzip", ["-p", zip.path, item.entry], { stdio: ["ignore", "pipe", "pipe"] });
  res.writeHead(200, {
    "content-type": mimeFor(item.entry),
    "cache-control": "private, max-age=3600",
    ...(download ? { "content-disposition": `attachment; filename="${item.entry.split("/").pop()}"` } : {}),
  });
  child.stdout.pipe(res);
  child.stderr.on("data", (chunk) => process.stderr.write(chunk));
  child.on("error", () => res.destroy());
}

createServer((req, res) => {
  try {
    const url = parseUrl(req);

    if (url.pathname === "/") {
      return sendText(res, 200, "Snapchat archive API is running. Open the Svelte app at http://127.0.0.1:5177/");
    }

    if (url.pathname === "/api/summary") {
      return sendJson(res, summary);
    }

    if (url.pathname === "/api/media") {
      return sendJson(res, media);
    }

    if (url.pathname === "/api/messages") {
      return sendJson(res, conversations);
    }

    if (url.pathname === "/api/json") {
      const index = Number(url.searchParams.get("index"));
      const item = jsonFiles[index];
      if (!item) return notFound(res);

      const zip = zipMeta[item.zipIndex];
      const raw = readZipText(zip.path, item.entry);
      const parsed = JSON.parse(raw);
      return sendJson(res, {
        file: item,
        summary: summarizeJson(parsed),
      });
    }

    const mediaMatch = url.pathname.match(/^\/media\/(\d+)$/);
    if (mediaMatch) {
      const item = media[Number(mediaMatch[1])];
      if (!item) return notFound(res);
      return streamZipEntry(res, item);
    }

    return notFound(res);
  } catch (error) {
    const message = error instanceof Error ? error.stack || error.message : String(error);
    console.error(message);
    sendText(res, 500, message);
  }
}).listen(port, "127.0.0.1", () => {
  console.log(`Snapchat archive viewer: http://127.0.0.1:${port}`);
  console.log(`Media: ${media.length}; JSON: ${jsonFiles.length}; entries: ${summary.totalEntries}`);
});
