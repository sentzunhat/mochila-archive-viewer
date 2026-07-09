#!/usr/bin/env npx tsx

import { execFileSync } from "child_process";
import { existsSync, readdirSync, readFileSync, statSync } from "fs";
import { extname, join, relative } from "path";

type JsonSummary = {
  path: string;
  kind: "array" | "object" | "other";
  recordCount?: number;
  keys: string[];
  nestedKeys: string[];
};

const args = process.argv.slice(2);
const inputFlag = args.indexOf("--input");
const input = inputFlag !== -1 ? args[inputFlag + 1] : args[0];

if (!input) {
  console.error("Usage: npx tsx inspect.ts --input <snapchat-export.zip-or-folder>");
  process.exit(1);
}

if (!existsSync(input)) {
  console.error(`Input not found: ${input}`);
  process.exit(1);
}

function isZip(path: string) {
  return extname(path).toLowerCase() === ".zip";
}

function listZipEntries(path: string) {
  const output = execFileSync("unzip", ["-Z1", path], { encoding: "utf8" });
  return output
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
}

function readZipEntry(path: string, entry: string) {
  return execFileSync("unzip", ["-p", path, entry], {
    encoding: "utf8",
    maxBuffer: 128 * 1024 * 1024,
  });
}

function walkFiles(root: string): string[] {
  const out: string[] = [];
  const stack = [root];

  while (stack.length > 0) {
    const current = stack.pop()!;
    const stat = statSync(current);

    if (stat.isDirectory()) {
      for (const entry of readdirSync(current)) {
        stack.push(join(current, entry));
      }
      continue;
    }

    if (stat.isFile()) {
      out.push(current);
    }
  }

  return out.sort();
}

function collectKeys(value: unknown, prefix = "", depth = 0): string[] {
  if (!value || typeof value !== "object" || depth > 1) {
    return [];
  }

  const keys: string[] = [];
  for (const key of Object.keys(value as Record<string, unknown>).slice(0, 80)) {
    const fullKey = prefix ? `${prefix}.${key}` : key;
    keys.push(fullKey);
    const child = (value as Record<string, unknown>)[key];
    if (child && typeof child === "object" && !Array.isArray(child)) {
      keys.push(...collectKeys(child, fullKey, depth + 1));
    }
  }

  return keys;
}

function summarizeJson(path: string, raw: string): JsonSummary {
  const parsed = JSON.parse(raw);
  const sample = Array.isArray(parsed) ? parsed[0] : parsed;
  const keys = collectKeys(sample).slice(0, 40);
  const nestedKeys = keys.filter((key) => key.includes("."));

  if (Array.isArray(parsed)) {
    return {
      path,
      kind: "array",
      recordCount: parsed.length,
      keys: keys.filter((key) => !key.includes(".")),
      nestedKeys,
    };
  }

  if (parsed && typeof parsed === "object") {
    return {
      path,
      kind: "object",
      keys: Object.keys(parsed as Record<string, unknown>).slice(0, 60),
      nestedKeys,
    };
  }

  return {
    path,
    kind: "other",
    keys: [],
    nestedKeys: [],
  };
}

function looksLikeMedia(path: string) {
  return /\.(jpe?g|png|heic|gif|mp4|mov|webm)$/i.test(path);
}

function main() {
  const zip = isZip(input);
  const files = zip ? listZipEntries(input) : walkFiles(input).map((file) => relative(input, file));
  const jsonFiles = files.filter((file) => file.toLowerCase().endsWith(".json"));
  const mediaFiles = files.filter(looksLikeMedia);

  console.log("Snapchat export inspection");
  console.log(`Input: ${input}`);
  console.log(`Mode : ${zip ? "zip" : "folder"}`);
  console.log("");
  console.log(`Files detected : ${files.length}`);
  console.log(`JSON files     : ${jsonFiles.length}`);
  console.log(`Media files    : ${mediaFiles.length}`);
  console.log("");

  if (jsonFiles.length === 0) {
    console.log("No JSON files found. If this is still downloading, inspect again after all zip parts finish.");
    return;
  }

  console.log("JSON summaries");
  console.log("--------------");

  for (const jsonFile of jsonFiles.slice(0, 80)) {
    try {
      const raw = zip ? readZipEntry(input, jsonFile) : readFileSync(join(input, jsonFile), "utf8");
      const summary = summarizeJson(jsonFile, raw);
      const count = summary.recordCount === undefined ? "" : ` records=${summary.recordCount}`;

      console.log(`\n${summary.path}`);
      console.log(`  kind=${summary.kind}${count}`);
      if (summary.keys.length > 0) {
        console.log(`  keys=${summary.keys.join(", ")}`);
      }
      if (summary.nestedKeys.length > 0) {
        console.log(`  nested=${summary.nestedKeys.slice(0, 20).join(", ")}`);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      console.log(`\n${jsonFile}`);
      console.log(`  error=${message}`);
    }
  }

  if (jsonFiles.length > 80) {
    console.log(`\nSkipped ${jsonFiles.length - 80} additional JSON files in this summary.`);
  }
}

main();
