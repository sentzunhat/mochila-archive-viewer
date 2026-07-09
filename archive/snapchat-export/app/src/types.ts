export type ArchiveSummary = {
  generatedAt: string;
  zipCount: number;
  totalZipBytes: number;
  totalEntries: number;
  mediaCount: number;
  jsonCount: number;
  htmlCount: number;
  zips: Array<{
    zipIndex: number;
    name: string;
    size: number;
    entries: number;
  }>;
  categories: Record<string, number>;
  years: Record<string, number>;
  types: Record<string, number>;
  jsonFiles: JsonFile[];
  htmlFiles: Array<{
    zipIndex: number;
    zip: string;
    entry: string;
  }>;
};

export type JsonFile = {
  zipIndex: number;
  zip: string;
  entry: string;
};

export type MediaItem = {
  id: number;
  zipIndex: number;
  zip: string;
  entry: string;
  category: string;
  date: string;
  year: string;
  type: "image" | "video" | "other";
  ext: string;
};

export type ConversationSummary = {
  id: string;
  title: string;
  messageCount: number;
  savedCount: number;
  mediaCount: number;
  lastCreated: string | null;
  messages: ChatMessage[];
};

export type ChatMessage = {
  from: string;
  content: string;
  mediaType: string;
  created: string;
  isSender: boolean;
  isSaved: boolean;
  mediaIds: string;
};

export type JsonSummary = {
  file: JsonFile;
  summary: {
    kind: string;
    records: number;
    keys: string[];
    childCounts?: Array<{
      key: string;
      type: string;
      records?: number;
    }>;
    sample: unknown;
  };
};
