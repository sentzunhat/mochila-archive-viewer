<script lang="ts">
  import { onMount } from "svelte";
  import {
    GetFrontendState,
    AvailableUsers,
    SelectUser,
    GetPlatformSnapshot,
    GetMedia,
    GetMediaPaginated,
    GetMediaCount,
    GetPlatformStats,
    GetAppSettings,
    SaveAppSettings,
    GetConversations,
    GetConversation,
    GetJSONPreview,
    GetMediaItem,
    SelectArchiveZips,
    IndexArchives,
    SaveProfile,
    LogoutProfile,
    AppVersion,
    CheckForUpdate,
  } from "../wailsjs/go/appshell/App.js";

  type ProviderCard = { id: string; name: string; status: string; description: string; supported: boolean };
  type Profile = { id: number; username: string; fullName: string; loggedIn: boolean };
  type FrontendState = { name: string; tagline: string; providers: ProviderCard[]; storePath: string; profile: Profile };
  type ArchiveFile = { path: string; name: string };
  type IndexSummary = {
    platform: string;
    mediaCount: number;
    zipCount: number;
    years: Record<string, number>;
    types: Record<string, number>;
    categories: Record<string, number>;
  };
  type MediaItem = {
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
  type ChatMessage = {
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
  type Conversation = {
    id: string;
    title: string;
    messageCount: number;
    savedCount: number;
    mediaCount: number;
    lastCreated: string;
    messages?: ChatMessage[];
  };
  type JsonFileRef = { zipIndex: number; zip: string; entry: string };
  type JSONPreview = {
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
  type PlatformSnapshot = { selected: ArchiveFile[]; summary: IndexSummary | null; media: MediaItem[]; jsonFiles: JsonFileRef[]; conversations: Conversation[] };
  type PlatformStatItem = { 
    id: string; name: string; status: string; 
    mediaCount: number; imageCount: number; videoCount: number; 
    zipCount: number; conversationCount: number; jsonFileCount: number; yearsFound: number 
  };
  type View = "gallery" | "today" | "messages" | "structure" | "data";
  type StructureSection = "paths" | "archives" | "summary";

  let appState: FrontendState = { name: "Mochila", tagline: "", providers: [], storePath: "", profile: { username: "", fullName: "", loggedIn: false } };
  let loading = true;
  let selecting = false;
  let indexing = false;
  let loadingConversation = false;
  let error: string | null = null;

  let activePlatform = "snapchat";
  let selected: ArchiveFile[] = [];
  let summary: IndexSummary | null = null;
  let media: MediaItem[] = [];
  let jsonFiles: JsonFileRef[] = [];
  let conversations: Conversation[] = [];
  let view: View = "gallery";
  let selectedYear = "all";
  let selectedCategory = "all";
  let selectedType = "all";
  let selectedConversationId: string | null = null;
  let selectedJSONIndex = 0;
  let jsonPreview: JSONPreview | null = null;
  let selectedMedia: MediaItem | null = null;
  let searchQuery = "";
  let profileOpen = false;
  let settingsOpen = false;
  let profileUsername = "";
  let profileFullName = "";
  let availableUsers: { id: number; username: string; fullName: string }[] = [];
  let searchInput: HTMLInputElement | undefined;
  let structureSection: StructureSection = "paths";
  $: pageSizeLabel = pageSize + " items per page";

  // Login & dashboard state
  let showLoginScreen: boolean = false;
  let loginUsername = "";
  let loginFullname = "";
  let authError = "";
  
  // Platform dashboard state  
  let showDashboard = false;
  let platformStatsList: PlatformStatItem[] = [];
  let selectedPlatform: string | null = null;
  
  // Infinite scroll & pagination state
  let paginatedMedia: MediaItem[] = [];
  let totalMediaCount: number = 0;
  let pageSize = 180;
  let currentOffset = 0;
  let isLoadingMore = false;
  let hasMoreMedia = true;
  
  // Infinite scroll observer ref
  let sentinelRef: HTMLElement | null = null;
  let infiniteObserver: IntersectionObserver | null = null;

  // ── Platform theming ──
  // Brand colors per platform; `accent` variables feed the custom CSS in
  // style.css, the Tailwind classes are picked up statically from this map.
  const platformThemes: Record<
    string,
    { name: string; accent: string; dark: string; soft: string; ink: string; bar: string; badge: string; card: string }
  > = {
    snapchat: {
      name: "Snapchat",
      accent: "#fffc00", dark: "#cfc600", soft: "#fffdd9", ink: "#181712",
      bar: "bg-snapchat",
      badge: "bg-snapchat text-archive-ink border-snapchat-dark",
      card: "hover:border-snapchat-dark",
    },
    instagram: {
      name: "Instagram",
      accent: "#dd2a7b", dark: "#8134af", soft: "#fdeef5", ink: "#ffffff",
      bar: "bg-gradient-to-r from-[#f58529] via-instagram to-[#515bd4]",
      badge: "bg-instagram-soft text-instagram border-instagram",
      card: "hover:border-instagram",
    },
    facebook: {
      name: "Facebook",
      accent: "#1877f2", dark: "#0e5fcb", soft: "#e9f2fe", ink: "#ffffff",
      bar: "bg-facebook",
      badge: "bg-facebook-soft text-facebook border-facebook",
      card: "hover:border-facebook",
    },
  };
  $: activeTheme = platformThemes[activePlatform] ?? platformThemes.snapchat;
  $: if (typeof document !== "undefined" && activeTheme) {
    const s = document.documentElement.style;
    s.setProperty("--accent", activeTheme.accent);
    s.setProperty("--accent-dark", activeTheme.dark);
    s.setProperty("--accent-soft", activeTheme.soft);
    s.setProperty("--accent-ink", activeTheme.ink);
  }

  const formatter = new Intl.NumberFormat();
  $: storeRoot = appState.storePath ? appState.storePath.replace("/database.sqlite", "") : "~/.mochila";
  $: providerRoot = `${storeRoot}/indexed/providers/${activePlatform}`;
  $: providerMediaRoot = `${providerRoot}/media`;
  $: providerSnapshotPath = `${providerRoot}/snapshot.json`;
  $: localDate = new Date();
  $: todayKey = `${String(localDate.getMonth() + 1).padStart(2, "0")}-${String(localDate.getDate()).padStart(2, "0")}`;

  $: years = summary ? Object.entries(summary.years).sort(([a], [b]) => b.localeCompare(a)) : [];
  $: maxYearCount = summary ? Math.max(...years.map(([, count]) => count), 1) : 0;
  $: categories = summary ? Object.entries(summary.categories).sort(([a], [b]) => a.localeCompare(b)) : [];
  $: types = summary ? Object.entries(summary.types).sort(([a], [b]) => a.localeCompare(b)) : [];
  // The gallery is always reached via a selected platform, and media is
  // always fetched pre-filtered from the backend (year/category/type/search
  // all narrow the SQL query — see currentMediaFilter/reloadFilteredMedia).
  $: visibleMedia = paginatedMedia;

  $: onThisDayMedia = media.filter((item) => item.date.slice(5, 10) === todayKey);
  $: selectedConversation =
    conversations.find((conversation) => conversation.id === selectedConversationId) ??
    conversations[0] ??
    null;

  function humanBytes(bytes: number) {
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
  // live archive, not a parsing gap). Label these distinctly from real
  // message content so they don't read as if the user actually typed "TEXT".
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
  function messageTypeLabel(type: string) {
    return messageTypeLabels[type] ?? (type || "No content saved");
  }

  function mediaName(item: MediaItem) {
    return item.entry.split("/").at(-1) ?? item.entry;
  }

  // Media is served directly over HTTP by the Go backend (appshell.ServeHTTP,
  // registered as the Wails asset server's fallback handler) rather than
  // fetched as a base64 data: URI over the Wails RPC bridge. The previous
  // approach read entire files (including multi-MB videos) into memory,
  // base64-inflated them (~33% larger), and serialized the result as a JSON
  // string over the same channel used for every other RPC call — with up to
  // 180 media items in a page, switching a filter while the old page's
  // fetches were still in flight visibly stalled the UI for several seconds.
  // A plain <img>/<video src> lets the browser's own HTTP stack handle
  // loading, concurrency (browsers cap ~6 per origin), and caching (see the
  // Cache-Control header ServeHTTP sets) for free.
  function mediaHttpSrc(platform: string, id: number) {
    // media_id is only unique per (platform, user_id) — the userId segment
    // isn't just informational, it's required for correctness: browsers
    // cache GET responses by URL, and without it, switching profiles mid
    // session could serve one user's cached photo at another user's
    // same-numbered id.
    return `/media/${platform}/${appState.profile.id}/${id}`;
  }

  // Reactive: load the selected conversation's messages once its summary
  // arrives. List-only conversations come back with `messages: null`
  // (Go serializes a nil slice as null, not []), so this must check falsy
  // rather than `.length === 0` — that comparison never matches null/undefined
  // and silently skipped the initial auto-load.
  $: if (
    !loading &&
    selectedConversationId &&
    !selectedConversation?.messages?.length &&
    !loadingConversation &&
    !loadedConversationIds.has(selectedConversationId) &&
    conversations.length > 0
  ) {
    openConversation(selectedConversationId);
  }

  // Media items opened from a chat message, keyed by id, so re-opening the
  // same one doesn't re-fetch its metadata.
  const messageMediaCache: Record<number, MediaItem> = {};

  async function openMessageMedia(id: number) {
    if (messageMediaCache[id]) {
      selectedMedia = messageMediaCache[id];
      return;
    }
    try {
      const item = await GetMediaItem(activePlatform, id);
      if (item) {
        messageMediaCache[id] = item;
        selectedMedia = item;
      }
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    }
  }

  async function loadPlatform(pid: string) {
    activePlatform = pid;
    error = null;

    const snapshot: PlatformSnapshot | null = await GetPlatformSnapshot(pid);
    selected = snapshot?.selected ?? [];
    summary = snapshot?.summary ?? null;
    media = snapshot?.media ?? [];
    jsonFiles = snapshot?.jsonFiles ?? [];
    conversations = snapshot?.conversations ?? [];
    selectedConversationId = conversations[0]?.id ?? null;
    // The snapshot's conversations are list-only (messages: null) — forget
    // any ids marked "loaded" by a previous snapshot fetch (e.g. the
    // pre-login onMount call), or the auto-load reactive will wrongly
    // believe this fresh, message-less snapshot is already populated.
    loadedConversationIds.clear();
    selectedJSONIndex = 0;
    jsonPreview = null;
    structureSection = "paths";
    selectedYear = "all";
    selectedCategory = "all";
    selectedType = "all";
    searchQuery = "";
    view = "gallery";
    if (jsonFiles.length > 0) {
      await openJSONFile(0);
    }
  }

    // ── Login/Logout handlers ──
  async function handleLogin() {
    if (!loginUsername.trim()) { authError = "Username is required"; return; }
    try {
      await SaveProfile(loginUsername.trim(), loginFullname.trim() || loginUsername.trim());
      showLoginScreen = false;
      showDashboard = true;
      await loadPlatformDashboard();
    } catch (e) {
      authError = "Login failed: " + String(e);
    }
  }

  async function handleLogout() {
    try {
      const nextProfile = await LogoutProfile();
      appState = { ...appState, profile: nextProfile };
    } catch (e) {}
    showLoginScreen = true;
    showDashboard = false;
    profileOpen = false;
    profileUsername = "";
    profileFullName = "";
    error = null;
    authError = "";
    dashboardNotice = "";
    loginUsername = "";
    loginFullname = "";
    paginatedMedia = [];
    selectedPlatform = null;
    currentOffset = 0;
    hasMoreMedia = true;
    // Conversation ids are not globally unique (e.g. "teamsnapchat" is a
    // system account shared by every user) — clear so the next login's
    // auto-load reactive doesn't skip a same-id conversation as "already
    // loaded" from the previous user's session.
    loadedConversationIds.clear();
  }

  // ── Dashboard loading ──
  async function loadPlatformDashboard() {
    try {
      const nextStats: PlatformStatItem[] = [];
      const platforms = ['snapchat', 'instagram', 'facebook'];
      for (const p of platforms) {
        let stats: any;
        try { stats = await GetPlatformStats(p); } catch(e) { continue; }
        if (stats) {
          nextStats.push({
            id: p, name: p.charAt(0).toUpperCase() + p.slice(1), status: stats.mediaCount > 0 ? 'indexed' : 'empty',
            mediaCount: Number(stats.mediaCount || 0), imageCount: Number(stats.imageCount || 0), videoCount: Number(stats.videoCount || 0),
            zipCount: Number(stats.zipCount || 0), conversationCount: Number(stats.conversationCount || 0), 
            jsonFileCount: Number(stats.jsonFileCount || 0), yearsFound: Number(stats.yearsFound || 0)
          });
        }
      }
      platformStatsList = nextStats;
    } catch (e) {
      console.error("Failed to load dashboard stats:", e);
    }
  }

  let dashboardNotice = "";
  let appVersion = "dev";
  let updateStatus: { available: boolean; latest: string; url: string } | null = null;

  async function selectPlatform(platform: string) {
    dashboardNotice = "";
    const prevPlatform = activePlatform;
    selectedPlatform = platform;
    showDashboard = false;
    paginatedMedia = [];
    totalMediaCount = 0;
    currentOffset = 0;
    hasMoreMedia = true;
    try {
      // Refresh the snapshot for the active user before paging media —
      // the onMount snapshot may belong to a previously active user.
      await loadPlatform(platform);
      await loadMediaBatch();
    } catch (e) {
      // e.g. platform not supported by the backend yet — stay on the dashboard.
      activePlatform = prevPlatform;
      selectedPlatform = null;
      showDashboard = true;
      error = null;
      const name = platformThemes[platform]?.name ?? platform;
      dashboardNotice = `${name} isn't supported yet — Snapchat is the only live provider for now.`;
    }
  }

  // ── Infinite scroll / pagination ──
  function currentMediaFilter() {
    return {
      Year: selectedYear,
      Category: selectedCategory,
      Type: selectedType,
      Search: searchQuery.trim(),
    };
  }

  // Bumped every time the filter changes and a fresh query is issued.
  // In-flight requests capture the generation they were issued under and
  // discard their result if a newer filter has since superseded them —
  // otherwise a slow "video only" response can land after a fast "all
  // types" reset and silently clobber it with stale data/counts.
  let mediaGeneration = 0;

  async function loadMediaBatch() {
    if (isLoadingMore || !hasMoreMedia || !selectedPlatform) return;
    const gen = mediaGeneration;
    isLoadingMore = true;

    try {
      const platform = selectedPlatform!;
      const filter = currentMediaFilter();

      // Get total count on the first page of a fresh filter.
      if (currentOffset === 0) {
        try {
          const count = await GetMediaCount(platform, filter);
          if (gen === mediaGeneration) totalMediaCount = count;
        } catch (e) {}
      }

      const items: MediaItem[] = (await GetMediaPaginated(platform, filter, currentOffset, pageSize)) ?? [];
      if (gen !== mediaGeneration) return;
      paginatedMedia = [...paginatedMedia, ...items];
      currentOffset += items.length;
      hasMoreMedia = items.length === pageSize;
    } catch (e) {
      console.error("Failed to load media:", e);
    } finally {
      isLoadingMore = false;
    }
  }

  // Re-run the paginated query from the top whenever a filter changes.
  // Debounced for search so we don't hit the DB on every keystroke.
  let filterReloadTimer: ReturnType<typeof setTimeout> | undefined;
  function reloadFilteredMedia(immediate = false) {
    if (!selectedPlatform) return;
    const run = () => {
      mediaGeneration += 1;
      paginatedMedia = [];
      currentOffset = 0;
      hasMoreMedia = true;
      // Force-unblock loadMediaBatch's in-flight guard so this newer
      // request starts immediately rather than waiting for a superseded
      // one to finish.
      isLoadingMore = false;
      loadMediaBatch();
    };
    if (immediate) {
      run();
      return;
    }
    clearTimeout(filterReloadTimer);
    filterReloadTimer = setTimeout(run, 300);
  }

  function triggerLoadMore() {
    if (!isLoadingMore && hasMoreMedia) {
      loadMediaBatch();
    }
  }

  // ── Infinite scroll observer ──
  function setupInfiniteScroll() {
    if (infiniteObserver) {
      infiniteObserver.disconnect();
    }
    
    if (!sentinelRef) return;
    
    infiniteObserver = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting && hasMoreMedia && !isLoadingMore) {
        triggerLoadMore();
      }
    }, { rootMargin: "400px" });
    
    infiniteObserver.observe(sentinelRef);
  }

  // Re-attach the observer whenever the sentinel enters/leaves the DOM
  // (it only renders while more media is available).
  $: if (sentinelRef !== undefined) {
    void sentinelRef;
    setupInfiniteScroll();
  }

  async function savePageSize(newSize: number) {
    pageSize = Math.max(30, Math.min(500, newSize));
    try { await SaveAppSettings({ pagesize: pageSize, loggedin: false }); } catch(e) {}
  }

  function handlePageSizeInput(event: Event) {
    const target = event.target as HTMLInputElement;
    savePageSize(Number(target.value));
  }

onMount(async () => {
    try {
      appState = await GetFrontendState();
      profileUsername = appState.profile.username ?? "";
      profileFullName = appState.profile.fullName ?? "";
      try { appVersion = await AppVersion(); } catch (e) {}
      try { updateStatus = await CheckForUpdate(); } catch (e) {}
      
      // Load user settings
      const settings = await GetAppSettings();
      if (settings) pageSize = Number(settings.pagesize) || 180;
      
      // Check login status and show appropriate screen
      if (!appState.profile.loggedIn) {
        showLoginScreen = true;
      } else {
        showDashboard = true;
        await loadPlatformDashboard();
      }
      await loadAvailableUsers();
      await loadPlatform(activePlatform);

      // Register keyboard shortcuts for search UX
      document.addEventListener("keydown", handleSearchKeyboard);
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    } finally {
      loading = false;
    }
  });

  function handleSearchKeyboard(event: KeyboardEvent) {
    // Only handle when not already focused on an input/textarea/select
    const target = event.target as HTMLElement;
    const tag = target?.tagName;
    if (tag === "INPUT" && target !== searchInput) return;
    if (tag === "TEXTAREA" || tag === "SELECT") return;

    // `/` focuses search from any view
    if (event.key === "/" && !event.metaKey && !event.ctrlKey && !event.altKey) {
      event.preventDefault();
      searchInput?.focus();
      return;
    }

    // `Escape` clears search and blurs (only when focused on search input)
    if (event.key === "Escape" && document.activeElement === searchInput) {
      event.preventDefault();
      searchQuery = "";
      searchInput?.blur();
      return;
    }

    // `Cmd/Ctrl+F` focuses search from any view
    if ((event.metaKey || event.ctrlKey) && event.key === "f") {
      event.preventDefault();
      searchInput?.focus();
      searchInput?.select();
    }
  }

  async function pickZips() {
    selecting = true;
    error = null;
    try {
      selected = (await SelectArchiveZips(activePlatform)) ?? [];
      summary = null;
      media = [];
      jsonFiles = [];
      conversations = [];
      selectedConversationId = null;
      selectedJSONIndex = 0;
      jsonPreview = null;
      structureSection = "paths";
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    } finally {
      selecting = false;
    }
  }

  async function indexSelected() {
    indexing = true;
    error = null;
    try {
      summary = await IndexArchives(activePlatform);
      media = (await GetMedia(activePlatform, "all")) ?? [];
      conversations = (await GetConversations(activePlatform)) ?? [];
      const snapshot = await GetPlatformSnapshot(activePlatform);
      jsonFiles = snapshot?.jsonFiles ?? [];
      selectedConversationId = conversations[0]?.id ?? null;
      selectedJSONIndex = 0;
      jsonPreview = null;
      selectedYear = "all";
      selectedCategory = "all";
      selectedType = "all";
      structureSection = "paths";

      // After indexing, refresh dashboard if visible
      if (showDashboard && selectedPlatform === activePlatform) {
        await loadPlatformDashboard();
      }
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    } finally {
      indexing = false;
    }
  }

  function setYear(year: string) {
    selectedYear = selectedYear === year ? "all" : year;
    reloadFilteredMedia(true);
  }

  // Conversation ids whose full message list has already been requested —
  // prevents the auto-load reactive from retrying a genuinely empty thread
  // on every reactive cycle.
  const loadedConversationIds = new Set<string>();

  async function openConversation(id: string) {
    loadingConversation = true;
    error = null;
    try {
      selectedConversationId = id;
      const full = await GetConversation(activePlatform, id);
      loadedConversationIds.add(id);
      if (full) {
        conversations = conversations.map((conversation) => (conversation.id === id ? full : conversation));
      }
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    } finally {
      loadingConversation = false;
    }
  }

  async function openJSONFile(index: number) {
    error = null;
    selectedJSONIndex = index;
    try {
      jsonPreview = await GetJSONPreview(activePlatform, index);
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    }
  }

  async function saveProfileForm() {
    try {
      const profile = await SaveProfile(profileUsername, profileFullName);
      appState = { ...appState, profile };
      if (profile.loggedIn) {
        showLoginScreen = false;
        showDashboard = true;
        await loadPlatformDashboard();
      }
      profileOpen = false;
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    }
  }

  async function switchUser(userId: number) {
    try {
      const profile = await SelectUser(userId);
      appState = { ...appState, profile };
      profileUsername = profile.username ?? "";
      profileFullName = profile.fullName ?? "";
      profileOpen = false;
      // Reload platform data for the new user
      loadedConversationIds.clear();
      await loadPlatform(activePlatform);
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    }
  }

  async function loadAvailableUsers() {
    try {
      availableUsers = await AvailableUsers();
    } catch (caught) {
      // Silently fail — user list is optional
    }
  }
</script>

<svelte:head>
  <title>Mochila</title>
</svelte:head>

{#if error}
  <main class="center">
    <section class="empty">
      <h1>Could not load the archive</h1>
      <p>{error}</p>
    </section>
  </main>
{:else if loading}
  <main class="center">
    <section class="empty">
      <h1>Loading Mochila...</h1>
      <p>Opening the local archive workspace.</p>
    </section>
  </main>
{:else if showLoginScreen}
  <!-- LOGIN SCREEN -->
  <main class="min-h-screen flex items-center justify-center bg-archive-bg px-4">
    <section class="w-full max-w-sm bg-archive-panel border border-archive-line rounded-2xl shadow-sm p-8">
      <div class="text-center mb-8">
        <div class="mx-auto mb-4 h-12 w-12 rounded-2xl bg-snapchat border border-snapchat-dark flex items-center justify-center text-2xl">🎒</div>
        <h1 class="text-2xl font-extrabold text-archive-ink m-0 mb-1">Mochila Archive Viewer</h1>
        <p class="text-sm text-archive-muted m-0">Personal data archive explorer</p>
      </div>

      {#if authError}
        <p class="mb-4 text-center text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">{authError}</p>
      {/if}

      <form on:submit|preventDefault={handleLogin} class="flex flex-col gap-4">
        <div>
          <label for="login-username" class="block mb-1 text-sm font-semibold text-archive-ink">Username</label>
          <input
            id="login-username"
            type="text"
            bind:value={loginUsername}
            placeholder="Enter your username"
            autocomplete="username"
            class="w-full px-3 py-2 rounded-lg border border-archive-line bg-white text-archive-ink placeholder:text-archive-muted/70 focus:outline-none focus:ring-2 focus:ring-snapchat-dark"
          />
        </div>

        <div>
          <label for="login-fullname" class="block mb-1 text-sm font-semibold text-archive-ink">Full Name (optional)</label>
          <input
            id="login-fullname"
            type="text"
            bind:value={loginFullname}
            placeholder="Enter your full name"
            autocomplete="name"
            class="w-full px-3 py-2 rounded-lg border border-archive-line bg-white text-archive-ink placeholder:text-archive-muted/70 focus:outline-none focus:ring-2 focus:ring-snapchat-dark"
          />
        </div>

        <button
          type="submit"
          class="mt-2 w-full py-2.5 rounded-lg bg-snapchat text-archive-ink font-extrabold border border-snapchat-dark hover:brightness-95 transition"
        >
          Sign In
        </button>
      </form>
    </section>
  </main>
{:else if showDashboard}
  <!-- PLATFORM DASHBOARD -->
  <main class="min-h-screen bg-archive-bg">
    <div class="max-w-4xl mx-auto p-8">
      <div class="flex justify-between items-center mb-8">
        <div>
          <h1 class="text-3xl font-extrabold text-archive-ink m-0 mb-1">Mochila Archive Viewer</h1>
          <p class="text-archive-muted m-0">Select a platform to explore your archive</p>
        </div>
        <button
          on:click={handleLogout}
          class="px-4 py-2 rounded-lg border border-archive-line bg-archive-panel text-archive-ink font-semibold hover:bg-white transition"
        >
          Logout
        </button>
      </div>

      {#if dashboardNotice}
        <p class="mb-6 rounded-lg border border-archive-line bg-archive-panel px-4 py-3 text-sm text-archive-muted">{dashboardNotice}</p>
      {/if}
      {#if platformStatsList.length === 0}
        <section class="empty">
          <h2>No platforms available yet</h2>
          <p>No archive data has been indexed for this profile. Continue to the explorer to add export zips.</p>
          <button class="load-more" on:click={() => { showDashboard = false; }}>Open archive explorer</button>
        </section>
      {/if}
      <div class="grid gap-6 grid-cols-[repeat(auto-fit,minmax(280px,1fr))]">
        {#each platformStatsList as stat}
          {@const theme = platformThemes[stat.id] ?? platformThemes.snapchat}
          <div
            tabindex="0"
            role="button"
            aria-label="Select {stat.name} platform"
            on:click={() => selectPlatform(stat.id)}
            on:keydown={(e) => e.key === "Enter" && selectPlatform(stat.id)}
            class="overflow-hidden rounded-2xl border border-archive-line bg-archive-panel cursor-pointer transition shadow-sm hover:shadow-md focus:outline-none focus:ring-2 focus:ring-archive-line {theme.card}"
          >
            <div class="h-1.5 {theme.bar}"></div>
            <div class="p-6">
              <div class="flex justify-between items-start mb-4">
                <h2 class="text-xl font-extrabold text-archive-ink capitalize m-0">{stat.name}</h2>
                <span
                  class="px-3 py-1 rounded-full text-xs font-semibold border {stat.status === 'indexed' ? theme.badge : 'bg-archive-bg text-archive-muted border-archive-line'}"
                >
                  {stat.status}
                </span>
              </div>

              <div class="grid grid-cols-2 gap-3">
                <div>
                  <div class="text-2xl font-extrabold text-archive-ink">{stat.mediaCount.toLocaleString()}</div>
                  <div class="text-xs text-archive-muted">Media Items</div>
                </div>
                <div>
                  <div class="text-2xl font-extrabold text-archive-ink">{stat.zipCount}</div>
                  <div class="text-xs text-archive-muted">Zips</div>
                </div>
                <div>
                  <div class="text-lg font-bold text-archive-ink">{stat.imageCount.toLocaleString()}</div>
                  <div class="text-xs text-archive-muted">Photos</div>
                </div>
                <div>
                  <div class="text-lg font-bold text-archive-ink">{stat.videoCount.toLocaleString()}</div>
                  <div class="text-xs text-archive-muted">Videos</div>
                </div>
              </div>

              {#if stat.conversationCount > 0}
                <div class="mt-4 pt-3 border-t border-archive-line text-sm text-archive-muted">
                  {stat.conversationCount.toLocaleString()} conversations · {stat.jsonFileCount.toLocaleString()} JSON files
                </div>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>
  </main>
{:else if !summary}
  <main class="center">
    <section class="empty">
      <h1>Indexing {activeTheme.name} export...</h1>
      <p>Your selected zips and indexed cache live in <code>{appState.storePath || "~/.mochila/database.sqlite"}</code>.</p>
      <p>
        {#if selected.length === 0}
          <button class="load-more" on:click={pickZips} disabled={selecting}>
            {selecting ? "Opening picker..." : "Choose export zip files"}
          </button>
        {:else}
          <button class="load-more" on:click={indexSelected} disabled={indexing}>
            {indexing ? "Indexing selected archives..." : `Index ${selected.length} zip${selected.length === 1 ? "" : "s"}`}
          </button>
        {/if}
      </p>
    </section>
  </main>
{:else}
  <header>
    <div class="topbar">
      <div>
        <p class="eyebrow flex items-center gap-2">
          <span class="inline-block h-2.5 w-2.5 rounded-full border border-archive-line {activeTheme.bar}"></span>
          {activeTheme.name} export loaded
        </p>
        <h1>Mochila</h1>
        <p class="subtitle">
          {summary.zipCount} zip files · {formatter.format(summary.mediaCount)} media items · stored in <code>{appState.storePath}</code>
        </p>
      </div>

      <nav class="tabs" aria-label="Archive views">
        <button class:active={view === "gallery"} on:click={() => (view = "gallery")}>
          Gallery
        </button>
        <button class:active={view === "today"} on:click={() => (view = "today")}>
          On This Day
        </button>
        <button class:active={view === "messages"} on:click={() => (view = "messages")}>
          Messages
        </button>
        <button class:active={view === "structure"} on:click={() => (view = "structure")}>
          Structure
        </button>
        <button
          class:active={view === "data"}
          on:click={async () => {
            view = "data";
            if (!jsonPreview && jsonFiles.length > 0) await openJSONFile(selectedJSONIndex);
          }}
        >
          Data Explorer
        </button>
        <button on:click={async () => { selectedPlatform = null; showDashboard = true; await loadPlatformDashboard(); }}>
          ← Dashboard
        </button>
        <button on:click={pickZips} disabled={selecting}>
          {selecting ? "Opening..." : "Add zips"}
        </button>
        <button on:click={() => (settingsOpen = true)}>⚙ Settings</button>
        <button on:click={() => (profileOpen = true)}>
          {appState.profile.loggedIn ? (appState.profile.fullName || appState.profile.username) : "Profile"}
        </button>
      </nav>
    </div>
  </header>

  <main>
    <section class="stats" aria-label="Archive summary">
      <article>
        <strong>{formatter.format(summary.mediaCount)}</strong>
        <span>media files</span>
      </article>
      <article>
        <strong>{formatter.format(summary.types.image ?? 0)}</strong>
        <span>images</span>
      </article>
      <article>
        <strong>{formatter.format(summary.types.video ?? 0)}</strong>
        <span>videos</span>
      </article>
      <article>
        <strong>{formatter.format(conversations.length)}</strong>
        <span>conversations</span>
      </article>
      <article>
        <strong>{selected.length}</strong>
        <span>cached zip sources</span>
      </article>
    </section>

    {#if view === "gallery"}
      <section class="controls" aria-label="Gallery filters">
        <div class="search-box">
          <input type="text" placeholder="Search files, categories, dates..." bind:value={searchQuery} bind:this={searchInput} on:input={() => reloadFilteredMedia()} />
          {#if searchQuery}
            <button class="clear-search" on:click={() => { searchQuery = ""; reloadFilteredMedia(true); }}>×</button>
          {/if}
        </div>

        <label>
          <span>Year</span>
          <select bind:value={selectedYear} on:change={() => setYear(selectedYear)}>
            <option value="all">All years</option>
            {#each years as [year, count]}
              <option value={year}>{year} ({formatter.format(count)})</option>
            {/each}
          </select>
        </label>
        <label>
          <span>Category</span>
          <select bind:value={selectedCategory} on:change={() => reloadFilteredMedia(true)}>
            <option value="all">All categories</option>
            {#each categories as [category, count]}
              <option value={category}>{category} ({formatter.format(count)})</option>
            {/each}
          </select>
        </label>
        <label>
          <span>Type</span>
          <select bind:value={selectedType} on:change={() => reloadFilteredMedia(true)}>
            <option value="all">All types</option>
            {#each types as [type, count]}
              <option value={type}>{type} ({formatter.format(count)})</option>
            {/each}
          </select>
        </label>
        <label>
          <span>Storage</span>
          <input value={appState.storePath} readonly />
        </label>
      </section>

      <section class="gallery-layout">
        <aside class="year-list" aria-label="Years">
          <button class:active={selectedYear === "all"} on:click={() => setYear("all")}>
            <strong>All</strong>
            <span class="bar"><span class="w-full"></span></span>
            <em>{formatter.format(summary.mediaCount)}</em>
          </button>
          {#each years as [year, count]}
            <button class:active={selectedYear === year} on:click={() => setYear(year)}>
              <strong>{year}</strong>
              <span class="bar"><span style={`width:${Math.max(4, (count / maxYearCount) * 100)}%`}></span></span>
              <em>{formatter.format(count)}</em>
            </button>
          {/each}
        </aside>

        <div class="gallery-panel">
          <div class="result-line">
            Showing {formatter.format(visibleMedia.length)} of {formatter.format(totalMediaCount)} matching files
          </div>

          {#if visibleMedia.length === 0 && !isLoadingMore}
            <div class="empty">No media matched those filters.</div>
          {:else}
            <div class="media-grid">
              {#each visibleMedia as item (item.id)}
                <article class="tile">
                  <button class="preview" on:click={() => (selectedMedia = item)} aria-label={`Open ${mediaName(item)}`}>
                    {#if item.type === "image"}
                      <img loading="lazy" src={mediaHttpSrc(activePlatform, item.id)} alt={mediaName(item)} />
                    {:else}
                      <video preload="metadata" controls muted src={mediaHttpSrc(activePlatform, item.id)} playsinline></video>
                    {/if}
                  </button>
                  <div class="tile-meta">
                    <strong title={item.entry}>{mediaName(item)}</strong>
                    <span>{item.date} · {item.category} · {item.type}</span>
                  </div>
                </article>
              {/each}
            </div>

            {#if hasMoreMedia}
              <button class="load-more" on:click={triggerLoadMore} disabled={isLoadingMore}>
                {isLoadingMore ? "Loading..." : `Load more (${paginatedMedia.length} of ${totalMediaCount.toLocaleString()})`}
              </button>
              <!-- Infinite scroll sentinel -->
              <div bind:this={sentinelRef} class="h-px"></div>
            {/if}
          {/if}
        </div>
      </section>
    {:else if view === "today"}
      <section class="today-layout">
        <div class="today-hero">
          <div>
            <p class="eyebrow">On this day</p>
            <h2>{todayKey}</h2>
            <p>{formatter.format(onThisDayMedia.length)} memories and chat media from this calendar day.</p>
          </div>
        </div>

        {#if onThisDayMedia.length === 0}
          <div class="empty">No exported Snapchat media landed on this date.</div>
        {:else}
          <div class="media-grid">
            {#each onThisDayMedia as item (item.id)}
              <article class="tile">
                <button class="preview" on:click={() => (selectedMedia = item)} aria-label={`Open ${mediaName(item)}`}>
                  {#if item.type === "image"}
                    <img loading="lazy" src={mediaHttpSrc(activePlatform, item.id)} alt={mediaName(item)} />
                  {:else}
                    <video preload="metadata" controls muted src={mediaHttpSrc(activePlatform, item.id)} playsinline></video>
                  {/if}
                </button>
                <div class="tile-meta">
                  <strong title={item.entry}>{mediaName(item)}</strong>
                  <span>{item.date} · {item.category} · {item.type}</span>
                </div>
              </article>
            {/each}
          </div>
        {/if}
      </section>
    {:else if view === "messages"}
      <section class="messages-layout">
        <aside class="conversation-list">
          {#each conversations as conversation}
            <button
              class:active={selectedConversation?.id === conversation.id}
              on:click={() => openConversation(conversation.id)}
            >
              <strong>{conversation.title}</strong>
              <span>
                {formatter.format(conversation.messageCount)} messages ·
                {formatter.format(conversation.savedCount)} saved ·
                {conversation.lastCreated ?? "no date"}
              </span>
            </button>
          {/each}
        </aside>

        <section class="message-panel">
          {#if loadingConversation}
            <div class="empty">Loading conversation...</div>
          {:else if !selectedConversation}
            <div class="empty">No chat history found in this export.</div>
          {:else}
            <div class="message-heading">
              <div>
                <h2>{selectedConversation.title}</h2>
                <p>
                  {formatter.format(selectedConversation.messageCount)} messages ·
                  {formatter.format(selectedConversation.mediaCount)} media references
                </p>
              </div>
            </div>

            <div class="messages">
              {#each selectedConversation.messages ?? [] as message}
                <article class:sent={message.isSender} class="message">
                  <div>
                    <strong>{message.from || (message.isSender ? "You" : selectedConversation.title)}</strong>
                    <span>{message.created}</span>
                  </div>
                  {#if message.content}
                    <p>{message.content}</p>
                  {:else if !message.mediaId}
                    <p class="italic text-archive-muted">{messageTypeLabel(message.mediaType)}</p>
                  {/if}
                  {#if message.mediaId != null}
                    <button
                      class="mt-2 block w-full max-w-[220px] overflow-hidden rounded-lg border border-archive-line bg-archive-bg"
                      on:click={() => openMessageMedia(message.mediaId)}
                      aria-label="Open attached media"
                    >
                      {#if message.linkedMediaType === "video"}
                        <video preload="metadata" muted class="block max-h-56 w-full object-cover" src={mediaHttpSrc(activePlatform, message.mediaId)}></video>
                      {:else}
                        <img loading="lazy" class="block max-h-56 w-full object-cover" src={mediaHttpSrc(activePlatform, message.mediaId)} alt="Attached media" />
                      {/if}
                    </button>
                  {/if}
                  {#if message.isSaved}
                    <small>saved</small>
                  {/if}
                </article>
              {/each}
            </div>
          {/if}
        </section>
      </section>
    {:else if view === "structure"}
      <section class="structure">
        <div class="structure-shell">
          <aside class="structure-nav" aria-label="Structure sections">
            <button class:active={structureSection === "paths"} on:click={() => (structureSection = "paths")}>Paths</button>
            <button class:active={structureSection === "archives"} on:click={() => (structureSection = "archives")}>Archives</button>
            <button class:active={structureSection === "summary"} on:click={() => (structureSection = "summary")}>Summary</button>
          </aside>

          <div class="structure-body">
            {#if structureSection === "paths"}
              <div class="structure-grid">
                <article class="structure-card">
                  <p class="eyebrow">Database</p>
                  <h2>Indexed metadata lives in SQLite</h2>
                  <p><code>{appState.storePath}</code></p>
                </article>
                <article class="structure-card">
                  <p class="eyebrow">Media cache</p>
                  <h2>Provider media cache</h2>
                  <p><code>{providerMediaRoot}</code></p>
                </article>
                <article class="structure-card">
                  <p class="eyebrow">Snapshot JSON</p>
                  <h2>Provider snapshot file</h2>
                  <p><code>{providerSnapshotPath}</code></p>
                </article>
                <article class="structure-card">
                  <p class="eyebrow">JSON explorer</p>
                  <h2>{formatter.format(jsonFiles.length)} JSON entries discovered</h2>
                  <p>Use Data Explorer if the provider export included raw JSON files.</p>
                </article>
              </div>
            {:else if structureSection === "archives"}
              <div class="zip-grid">
                {#each selected as zip}
                  <article>
                    <strong>{zip.name}</strong>
                    <span>{zip.path}</span>
                  </article>
                {/each}
              </div>
            {:else}
              <pre>{JSON.stringify(summary, null, 2)}</pre>
            {/if}
          </div>
        </div>
      </section>
    {:else}
      <section class="messages-layout">
        <aside class="conversation-list">
          {#each jsonFiles as item, index}
            <button
              class:active={selectedJSONIndex === index}
              on:click={() => openJSONFile(index)}
            >
              <strong>{item.entry.split("/").at(-1) ?? item.entry}</strong>
              <span>{item.zip} · {item.entry}</span>
            </button>
          {/each}
        </aside>

        <section class="message-panel">
          {#if jsonFiles.length === 0}
            <div class="empty">No JSON files were discovered in the selected export zips.</div>
          {:else if !jsonPreview}
            <div class="empty">Choose a JSON file to inspect its structure and contents.</div>
          {:else}
            <div class="message-heading">
              <div>
                <h2>{jsonPreview.entry.split("/").at(-1) ?? jsonPreview.entry}</h2>
                <p>
                  {jsonPreview.zip} · {jsonPreview.topLevel} ·
                  {formatter.format(jsonPreview.itemCount)} items
                </p>
              </div>
            </div>

            <section class="structure-grid">
              <article class="structure-card">
                <p class="eyebrow">Top-level keys</p>
                <h2>{jsonPreview.keys.length === 0 ? "No object keys" : jsonPreview.keys.join(", ")}</h2>
              </article>
              <article class="structure-card">
                <p class="eyebrow">Snapshot file</p>
                <h2><code>{jsonPreview.storagePath}</code></h2>
              </article>
            </section>

            <pre>{jsonPreview.prettyJson}</pre>
          {/if}
        </section>
      </section>
    {/if}
  </main>

  {#if selectedMedia}
    <div class="modal-backdrop" role="presentation" on:click={() => (selectedMedia = null)}>
      <div class="media-modal" role="dialog" aria-modal="true" aria-label={mediaName(selectedMedia)} tabindex="-1" on:click|stopPropagation on:keydown={(e) => e.key === "Escape" && (selectedMedia = null)}>
        <button class="close-button" on:click={() => (selectedMedia = null)} aria-label="Close">×</button>
        <div class="modal-media">
          {#if selectedMedia.type === "image"}
            <img src={mediaHttpSrc(activePlatform, selectedMedia.id)} alt={mediaName(selectedMedia)} />
          {:else}
            <video src={mediaHttpSrc(activePlatform, selectedMedia.id)} controls autoplay>
              <track kind="captions" />
            </video>
          {/if}
        </div>
        <aside class="modal-meta">
          <p class="eyebrow">Media details</p>
          <h2>{mediaName(selectedMedia)}</h2>
          <dl>
            <div><dt>Date</dt><dd>{selectedMedia.date}</dd></div>
            <div><dt>Type</dt><dd>{selectedMedia.type}</dd></div>
            <div><dt>Category</dt><dd>{selectedMedia.category}</dd></div>
            <div><dt>Extension</dt><dd>{selectedMedia.ext}</dd></div>
            <div><dt>Source zip</dt><dd>{selectedMedia.zip}</dd></div>
            <div><dt>Archive path</dt><dd>{selectedMedia.entry}</dd></div>
          </dl>
        </aside>
      </div>
    </div>
  {/if}
{/if}

{#if settingsOpen}
  <div class="modal-backdrop" role="presentation" on:click={() => (settingsOpen = false)}>
    <div class="settings-modal" role="dialog" aria-modal="true" on:click|stopPropagation on:keydown|stopPropagation>
      <div class="settings-head">
        <div>
          <p class="eyebrow">Settings</p>
          <h2>Application preferences</h2>
        </div>
        <button class="close-button" on:click={() => (settingsOpen = false)} aria-label="Close">×</button>
      </div>
      
      <section class="settings-section">
        <h3>About</h3>
        <dl class="settings-list">
          <div><dt>Version</dt><dd>Mochila {appVersion}</dd></div>
          <div><dt>Backend</dt><dd>Go with Wails v2</dd></div>
          <div><dt>Storage</dt><dd>{appState.storePath || "~/.mochila/database.sqlite"}</dd></div>
        </dl>
        {#if updateStatus?.available}
          <p class="settings-hint">
            <a href={updateStatus.url} target="_blank" rel="noreferrer">{updateStatus.latest} available — download</a>
          </p>
        {/if}
        <button class="secondary-button" on:click={async () => { try { updateStatus = await CheckForUpdate(); } catch (e) {} }}>
          Check for updates
        </button>
      </section>
      
      <section class="settings-section">
        <h3>Data Management</h3>
        <dl class="settings-list">
          <div><dt>Total media items</dt><dd>{summary ? formatter.format(summary.mediaCount) : "N/A"}</dd></div>
          <div><dt>Archive files indexed</dt><dd>{summary ? summary.zipCount : 0}</dd></div>
          <div><dt>Conversations</dt><dd>{formatter.format(conversations.length)}</dd></div>
        </dl>
        <p class="settings-hint">Indexed data is stored locally in SQLite.</p>
      </section>

      <section class="settings-section">
        <h3>Display</h3>
        <div class="mb-4">
          <label for="page-size-slider" class="flex flex-col gap-1">
            <span>Items per page (gallery)</span>
            <input
              id="page-size-slider"
              type="range"
              min="30"
              max="500"
              step="10"
              value={pageSize}
              on:input={handlePageSizeInput}
              class="w-full max-w-[200px]"
            />
            <span class="text-sm text-archive-muted">{pageSizeLabel}</span>
          </label>
        </div>
      </section>
    </div>
  </div>
{/if}

{#if profileOpen}
  <div class="modal-backdrop" role="presentation" on:click={() => (profileOpen = false)}>
    <div class="profile-modal" role="dialog" aria-modal="true" on:click|stopPropagation on:keydown|stopPropagation>
      <div class="profile-head">
        <div>
          <p class="eyebrow">Profile</p>
          <h2>{appState.profile.loggedIn ? "Your archive profile" : "Create a simple profile"}</h2>
        </div>
        <button class="close-button" on:click={() => { profileOpen = false; }} aria-label="Close">×</button>
      </div>
      {#if availableUsers.length > 0}
        <p class="section-label">Switch profile</p>
        <div class="user-list">
          {#each availableUsers as user}
            <button class="user-pill" on:click={() => switchUser(user.id)}>
              <span class="user-name">{user.fullName || user.username}</span>
              {#if appState.profile.id === user.id && appState.profile.loggedIn}
                <span class="current-badge">you</span>
              {/if}
            </button>
          {/each}
        </div>
      {/if}
      <label>
        <span>Username</span>
        <input bind:value={profileUsername} placeholder="archivekeeper" />
      </label>
      <label>
        <span>Full name</span>
        <input bind:value={profileFullName} placeholder="Diego Beltran" />
      </label>
      <div class="profile-actions">
        <button class="load-more" on:click={saveProfileForm}>Save profile</button>
        {#if appState.profile.loggedIn}
          <button class="secondary-button" on:click={() => { profileUsername = ""; profileFullName = ""; }}>New profile</button>
          <button class="secondary-button" on:click={handleLogout}>Logout</button>
        {/if}
      </div>
    </div>
  </div>
{/if}
