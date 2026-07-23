export const formatter = new Intl.NumberFormat();

export function humanBytes(bytes: number): string {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = bytes;
  let unit = 0;
  while (size > 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

// Snapchat's export often has no retained text for ephemeral chat types
// (expired snaps, stories, location shares, etc — confirmed against the
// live archive, not a parsing gap).
const messageTypeLabels: Record<string, string> = {
  TEXT: "No text saved",
  MEDIA: "Media (no caption)",
  SHARE: "Shared content",
  SHARESAVEDSTORY: "Shared story",
  STICKER: "Sticker",
  STATUS: "Status update",
  NOTE: "Voice note",
  LOCATION: "Location share",
  STATUSERASEDMESSAGE: "Message expired",
};

export function messageTypeLabel(type: string): string {
  return messageTypeLabels[type] ?? (type || "No content saved");
}

export function mediaName(entry: string): string {
  return entry.split("/").at(-1) ?? entry;
}
