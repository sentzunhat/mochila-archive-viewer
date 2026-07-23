export type ProviderCard = { id: string; name: string; status: string; description: string; supported: boolean };
export type Profile = { id: number; username: string; fullName: string; loggedIn: boolean };
export type FrontendState = { name: string; tagline: string; providers: ProviderCard[]; storePath: string; profile: Profile };
export type ArchiveFile = { path: string; name: string };
export type IndexSummary = {
  platform: string;
  mediaCount: number;
  zipCount: number;
  years: Record<string, number>;
  types: Record<string, number>;
  categories: Record<string, number>;
};
export type MediaItem = {
  id: number;
  zipIndex: number;
  zip: string;
  entry: string;
  category: string;
  date: string;
  year: string;
  type: string;
  ext: string;
  localPath?: string;
};
export type ChatMessage = {
  from: string;
  content: string;
  mediaType: string;
  created: string;
  isSender: boolean;
  isSaved: boolean;
  mediaIds: string;
  mediaId?: number;
  linkedMediaType?: string;
};
export type Conversation = {
  id: string;
  title: string;
  messageCount: number;
  savedCount: number;
  mediaCount: number;
  lastCreated: string;
  messages?: ChatMessage[];
};
export type JsonFileRef = { zipIndex: number; zip: string; entry: string };
export type JSONPreview = {
  entry: string;
  zip: string;
  topLevel: string;
  keys: string[];
  itemCount: number;
  prettyJson: string;
  storagePath: string;
  childCounts?: { key: string; type: string; records?: number }[];
  sampleJson?: string;
};
export type PlatformSnapshot = {
  selected: ArchiveFile[];
  summary: IndexSummary | null;
  media: MediaItem[];
  jsonFiles: JsonFileRef[];
  conversations: Conversation[];
};
export type PlatformStatItem = {
  id: string; name: string; status: string;
  mediaCount: number; imageCount: number; videoCount: number;
  zipCount: number; conversationCount: number; jsonFileCount: number; yearsFound: number;
};
export type View = "gallery" | "today" | "messages" | "structure" | "data";
export type StructureSection = "paths" | "archives" | "summary";
export type Theme = { name: string; accent: string; dark: string; soft: string; ink: string; bar: string; badge: string; card: string };
export type UpdateStatus = { available: boolean; latest: string; url: string };
