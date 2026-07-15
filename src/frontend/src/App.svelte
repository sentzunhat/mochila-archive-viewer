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
    GetMediaSource,
    SelectArchiveZips,
    IndexArchives,
    SaveProfile,
    LogoutProfile,
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
  let visibleLimit = 180;
  let searchQuery = "";
  let mediaSources: Record<number, string> = {};
  let mediaLoading: Record<number, "pending" | "resolved" | "failed"> = {};
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
  $: filteredMedia = media.filter((item) => {
    if (selectedYear !== "all" && item.year !== selectedYear) return false;
    if (selectedCategory !== "all" && item.category !== selectedCategory) return false;
    if (selectedType !== "all" && item.type !== selectedType) return false;
    return true;
  });
  $: searchableMedia = searchQuery
    ? filteredMedia.filter(m => m.entry.toLowerCase().includes(searchQuery.toLowerCase()) || m.category.toLowerCase().includes(searchQuery.toLowerCase()) || m.date.includes(searchQuery))
    : filteredMedia;
  // Use paginated media when in infinite scroll mode, otherwise original slicing
  $: visibleMedia = selectedPlatform ? paginatedMedia : searchableMedia.slice(0, visibleLimit);
  
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

  function mediaName(item: MediaItem) {
    return item.entry.split("/").at(-1) ?? item.entry;
  }

  function sourceFor(item: MediaItem) {
    if (mediaSources[item.id]) return mediaSources[item.id];
    return "";
  }

  async function ensureMediaSources(items: MediaItem[]) {
    const pending = items.filter((item) => {
      if (mediaSources[item.id]) return false;
      if (mediaLoading[item.id] === "resolved") return false;
      if (mediaLoading[item.id] === "pending") return false;
      return true;
    }).slice(0, 50);
    if (pending.length === 0) return;

    const updates: Record<number, string> = {};
    // Mark all as pending first
    for (const item of pending) {
      mediaLoading[item.id] = "pending";
    }
    mediaLoading = { ...mediaLoading };

    await Promise.all(
      pending.map(async (item) => {
        try {
          const source = await GetMediaSource(activePlatform, item.id);
          if (source && source.length > 0) {
            updates[item.id] = source;
            mediaLoading[item.id] = "resolved";
          } else {
            mediaLoading[item.id] = "failed";
          }
        } catch (caught) {
          error = caught instanceof Error ? caught.message : String(caught);
          mediaLoading[item.id] = "failed";
        }
      }),
    );
    mediaSources = { ...mediaSources, ...updates };
  }

  // Reactive: auto-fetch sources for visible media (gallery view, filter changes)
  $: if (!loading && summary && visibleMedia.length > 0) {
    const unsourced = visibleMedia.filter(
      (m) => !mediaSources[m.id] && mediaLoading[m.id] !== "resolved" && mediaLoading[m.id] !== "pending",
    );
    if (unsourced.length > 0) {
      ensureMediaSources(unsourced);
    }
  }

  // Reactive: auto-fetch sources for today view items
  $: if (!loading && summary && onThisDayMedia.length > 0) {
    const unsourced = onThisDayMedia.filter(
      (m) => !mediaSources[m.id] && mediaLoading[m.id] !== "resolved" && mediaLoading[m.id] !== "pending",
    );
    if (unsourced.length > 0) {
      ensureMediaSources(unsourced);
    }
  }

  // Reactive: auto-fetch sources when conversations become visible
  $: if (!loading && selectedConversationId && selectedConversation?.messages?.length === 0 && conversations.length > 0) {
    openConversation(selectedConversationId);
  }

  async function retryFailedSources(items: MediaItem[]) {
    const failed = items.filter((item) => mediaLoading[item.id] === "failed");
    if (failed.length === 0) return;
    for (const item of failed) {
      try {
        const source = await GetMediaSource(activePlatform, item.id);
        if (source && source.length > 0) {
          mediaSources[item.id] = source;
          mediaLoading[item.id] = "resolved";
        } else {
          mediaLoading[item.id] = "failed";
        }
      } catch {
        mediaLoading[item.id] = "failed";
      }
    }
    mediaSources = { ...mediaSources };
    mediaLoading = { ...mediaLoading };
  }

  async function loadPlatform(pid: string) {
    activePlatform = pid;
    error = null;
    mediaSources = {};

    const snapshot: PlatformSnapshot | null = await GetPlatformSnapshot(pid);
    selected = snapshot?.selected ?? [];
    summary = snapshot?.summary ?? null;
    media = snapshot?.media ?? [];
    jsonFiles = snapshot?.jsonFiles ?? [];
    conversations = snapshot?.conversations ?? [];
    selectedConversationId = conversations[0]?.id ?? null;
    selectedJSONIndex = 0;
    jsonPreview = null;
    structureSection = "paths";
    selectedYear = "all";
    selectedCategory = "all";
    selectedType = "all";
    visibleLimit = 180;
    view = summary ? "gallery" : "gallery";
    if (jsonFiles.length > 0) {
      await openJSONFile(0);
    }
    await ensureMediaSources(media.slice(0, 180));
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
    try { await LogoutProfile(); } catch(e) {}
    showLoginScreen = true;
    showDashboard = false;
    authError = "";
    loginUsername = "";
    loginFullname = "";
    paginatedMedia = [];
    selectedPlatform = null;
    currentOffset = 0;
    hasMoreMedia = true;
    mediaSources = {};
    mediaLoading = {};
  }

  // ── Dashboard loading ──
  async function loadPlatformDashboard() {
    try {
      platformStatsList = [];
      const platforms = ['snapchat', 'instagram', 'facebook'];
      for (const p of platforms) {
        let stats: any;
        try { stats = await GetPlatformStats(p); } catch(e) { continue; }
        if (stats) {
          platformStatsList.push({
            id: p, name: p.charAt(0).toUpperCase() + p.slice(1), status: stats.mediaCount > 0 ? 'indexed' : 'empty',
            mediaCount: Number(stats.mediaCount || 0), imageCount: Number(stats.imageCount || 0), videoCount: Number(stats.videoCount || 0),
            zipCount: Number(stats.zipCount || 0), conversationCount: Number(stats.conversationCount || 0), 
            jsonFileCount: Number(stats.jsonFileCount || 0), yearsFound: Number(stats.yearsFound || 0)
          });
        }
      }
    } catch (e) {
      console.error("Failed to load dashboard stats:", e);
    }
  }

  async function selectPlatform(platform: string) {
    selectedPlatform = platform;
    paginatedMedia = [];
    totalMediaCount = 0;
    currentOffset = 0;
    hasMoreMedia = true;
    mediaSources = {};
    mediaLoading = {};
    await loadMediaBatch();
  }

  // ── Infinite scroll / pagination ──
  async function loadMediaBatch() {
    if (isLoadingMore || !hasMoreMedia || !selectedPlatform) return;
    isLoadingMore = true;
    
    try {
      const platform = selectedPlatform!;
      
      // Get total count first time
      if (currentOffset === 0) {
        try { 
          totalMediaCount = await GetMediaCount(platform, ""); 
        } catch(e) {}
      }
      
      const newItems: MediaItem[] = [];
      let batchOffset = currentOffset;
      let keepLoading = true;
      
      while (keepLoading) {
        const items = await GetMediaPaginated(
          platform, "",
          batchOffset,
          pageSize
        );
        
        if (!items || items.length === 0) break;
        
        newItems.push(...items);
        batchOffset += items.length;
        
        if (items.length < pageSize) break;
      }
      
      paginatedMedia.push(...newItems);
      currentOffset = batchOffset;
      hasMoreMedia = totalMediaCount > 0 ? currentOffset < totalMediaCount : currentOffset === 0;
      
      // Reset observer target
      setupInfiniteScroll();
    } catch (e) {
      console.error("Failed to load media:", e);
    } finally {
      isLoadingMore = false;
    }
  }

  function triggerLoadMore() {
    if (!isLoadingMore && hasMoreMedia) {
      currentOffset = paginatedMedia.length;
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
      mediaSources = {};
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
      mediaSources = {};
      await ensureMediaSources(media.slice(0, 60));
      
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

  async function setYear(year: string) {
    selectedYear = selectedYear === year ? "all" : year;
    visibleLimit = 180;
    media = (await GetMedia(activePlatform, selectedYear)) ?? [];
    mediaSources = {};
    await ensureMediaSources(media.slice(0, 60));
  }

  async function openConversation(id: string) {
    loadingConversation = true;
    error = null;
    try {
      selectedConversationId = id;
      const full = await GetConversation(activePlatform, id);
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

  async function loadMore() {
    visibleLimit += 180;
    await ensureMediaSources(visibleMedia);
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

  async function logoutProfile() {
    try {
      const nextProfile = await LogoutProfile();
      appState = { ...appState, profile: nextProfile };
      profileUsername = "";
      profileFullName = "";
      showLoginScreen = true;
      showDashboard = false;
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
      mediaSources = {};
      mediaLoading = {};
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
  <main class="center">
    <section class="login-card">
      <div style="text-align:center;margin-bottom:2rem;">
        <h1 style="font-size:2rem;margin-bottom:0.5rem;">Mochila Archive Viewer</h1>
        <p class="subtitle" style="max-width:300px;margin:0 auto;">Snapchat Data Explorer</p>
      </div>
      
      {#if authError}
        <p style="color:#ef4444;margin-bottom:1rem;text-align:center;">{authError}</p>
      {/if}
      
      <form on:submit|preventDefault={handleLogin}>
        <div class="form-group" style="margin-bottom:1rem;">
          <label for="login-username" style="display:block;margin-bottom:0.25rem;font-weight:600;">Username</label>
          <input 
            id="login-username"
            type="text" 
            bind:value={loginUsername}
            placeholder="Enter your username"
            autocomplete="username"
            style="width:100%;padding:0.5rem;border:1px solid #374151;border-radius:0.375rem;background:#1f2937;color:white;"
          />
        </div>
        
        <div class="form-group" style="margin-bottom:1.5rem;">
          <label for="login-fullname" style="display:block;margin-bottom:0.25rem;font-weight:600;">Full Name (optional)</label>
          <input 
            id="login-fullname"
            type="text" 
            bind:value={loginFullname}
            placeholder="Enter your full name"
            autocomplete="name"
            style="width:100%;padding:0.5rem;border:1px solid #374151;border-radius:0.375rem;background:#1f2937;color:white;"
          />
        </div>
        
        <button type="submit" class="btn-primary" style="width:100%;padding:0.75rem;font-size:1rem;">
          Sign In
        </button>
      </form>
    </section>
  </main>
{:else if showDashboard}
  <!-- PLATFORM DASHBOARD -->
  <main class="dashboard-main">
    <div style="max-width:960px;margin:0 auto;padding:2rem;">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:2rem;">
        <div>
          <h1 style="font-size:1.875rem;margin-bottom:0.25rem;">Mochila Archive Viewer</h1>
          <p class="subtitle">Select a platform to explore your archive</p>
        </div>
        <button on:click={handleLogout} class="btn-secondary" style="padding:0.5rem 1rem;">
          Logout
        </button>
      </div>
      
      <div class="platform-grid" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:1.5rem;">
        {#each platformStatsList as stat}
          <div 
            class="platform-card" 
            tabindex="0"
            role="button"
            aria-label="Select {stat.name} platform"
            on:click={() => selectPlatform(stat.id)}
            on:keydown={(e) => e.key === "Enter" && selectPlatform(stat.id)}
            style="background:#1f2937;border:1px solid #374151;border-radius:0.75rem;padding:1.5rem;cursor:pointer;transition:all 0.2s;"
          >
            <div style="display:flex;justify-content:space-between;align-items:start;margin-bottom:1rem;">
              <h2 style="font-size:1.25rem;text-transform:capitalize;">{stat.name}</h2>
              <span class="badge" style="padding:0.25rem 0.75rem;border-radius:9999px;font-size:0.75rem;{ stat.status === 'indexed' ? 'background:#065f46;color:#6ee7b7;' : 'background:#374151;color:#9ca3af;' }">
                {stat.status}
              </span>
            </div>
            
            <div style="display:grid;grid-template-columns:repeat(2,1fr);gap:0.75rem;margin-top:1rem;">
              <div>
                <div style="font-size:1.5rem;font-weight:700;color:#f3f4f6;">{stat.mediaCount.toLocaleString()}</div>
                <div style="font-size:0.75rem;color:#9ca3af;">Media Items</div>
              </div>
              <div>
                <div style="font-size:1.25rem;font-weight:600;color:#f3f4f6;">{stat.zipCount}</div>
                <div style="font-size:0.75rem;color:#9ca3af;">Zips</div>
              </div>
              <div>
                <div style="font-size:1rem;font-weight:600;color:#60a5fa;">{stat.imageCount.toLocaleString()}</div>
                <div style="font-size:0.75rem;color:#9ca3af;">Photos</div>
              </div>
              <div>
                <div style="font-size:1rem;font-weight:600;color:#f87171;">{stat.videoCount.toLocaleString()}</div>
                <div style="font-size:0.75rem;color:#9ca3af;">Videos</div>
              </div>
            </div>
            
            {#if stat.conversationCount > 0}
              <div style="margin-top:1rem;padding-top:0.75rem;border-top:1px solid #374151;font-size:0.875rem;color:#9ca3af;">
                {stat.conversationCount.toLocaleString()} conversations · {stat.jsonFileCount.toLocaleString()} JSON files
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  </main>
{:else if !summary}
  <main class="center">
    <section class="empty">
      <h1>Indexing Snapchat export...</h1>
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
        <p class="eyebrow">Snapchat export loaded</p>
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
          <input type="text" placeholder="Search files, categories, dates..." bind:value={searchQuery} bind:this={searchInput} />
          {#if searchQuery}
            <button class="clear-search" on:click={() => (searchQuery = "")}>×</button>
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
          <select bind:value={selectedCategory} on:change={() => (visibleLimit = 180)}>
            <option value="all">All categories</option>
            {#each categories as [category, count]}
              <option value={category}>{category} ({formatter.format(count)})</option>
            {/each}
          </select>
        </label>
        <label>
          <span>Type</span>
          <select bind:value={selectedType} on:change={() => (visibleLimit = 180)}>
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
            <span class="bar"><span style="width:100%"></span></span>
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
            Showing {formatter.format(visibleMedia.length)} of {formatter.format(filteredMedia.length)} matching files
          </div>

          {#if filteredMedia.length === 0}
            <div class="empty">No media matched those filters.</div>
          {:else}
            <div class="media-grid">
              {#each visibleMedia as item (item.id)}
                <article class="tile">
                  <button class="preview" on:click={() => (selectedMedia = item)} aria-label={`Open ${mediaName(item)}`}>
                    {#if item.type === "image"}
                      {#if sourceFor(item)}
                        <img loading="lazy" src={sourceFor(item)} alt={mediaName(item)} />
                      {:else}
                        <div class="placeholder">Loading image…</div>
                      {/if}
                    {:else if sourceFor(item)}
                      <video preload="metadata" controls muted src={sourceFor(item)} playsinline></video>
                    {:else}
                      <div class="placeholder">Loading video…</div>
                    {/if}
                  </button>
                  <div class="tile-meta">
                    <strong title={item.entry}>{mediaName(item)}</strong>
                    <span>{item.date} · {item.category} · {item.type}</span>
                  </div>
                </article>
              {/each}
            </div>

            {#if visibleMedia.length < filteredMedia.length}
              <button class="load-more" on:click={triggerLoadMore} disabled={isLoadingMore}>
                {isLoadingMore ? "Loading..." : `Load more (${paginatedMedia.length} of ${totalMediaCount.toLocaleString()})`}
              </button>
              <!-- Infinite scroll sentinel -->
              <div bind:this={sentinelRef} style="height:1px;"></div>
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
                  {#if item.type === "image" && sourceFor(item)}
                    <img loading="lazy" src={sourceFor(item)} alt={mediaName(item)} />
                  {:else if sourceFor(item)}
                    <video preload="metadata" controls muted src={sourceFor(item)} playsinline></video>
                  {:else}
                    <div class="placeholder">Loading…</div>
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
                  <p>{message.content || message.mediaType || "Media / attachment"}</p>
                  {#if message.isSaved || message.mediaIds}
                    <small>
                      {message.isSaved ? "saved" : ""}
                      {message.mediaIds ? ` media: ${message.mediaIds}` : ""}
                    </small>
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
      <div class="media-modal" role="dialog" aria-modal="true" aria-label={mediaName(selectedMedia)} tabindex="-1" on:keydown={(e) => e.key === "Escape" && (selectedMedia = null)}>
        <button class="close-button" on:click={() => (selectedMedia = null)} aria-label="Close">×</button>
        <div class="modal-media">
          {#if selectedMedia.type === "image" && sourceFor(selectedMedia)}
            <img src={sourceFor(selectedMedia)} alt={mediaName(selectedMedia)} />
          {:else if sourceFor(selectedMedia)}
            <video src={sourceFor(selectedMedia)} controls autoplay>
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


// Page size for gallery display

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
          <div><dt>Version</dt><dd>Mochila v1.0.0</dd></div>
          <div><dt>Backend</dt><dd>Go 1.26 with Wails v2</dd></div>
          <div><dt>Storage</dt><dd>{appState.storePath || "~/.mochila/database.sqlite"}</dd></div>
        </dl>
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
        <div class="setting-row" style="margin-bottom:1rem;">
          <label for="page-size-slider" style="display:flex;flex-direction:column;gap:0.25rem;">
            <span>Items per page (gallery)</span>
            <input 
              id="page-size-slider"
              type="range" 
              min="30" 
              max="500" 
              step="10" 
              value={pageSize}
              on:input={handlePageSizeInput}
              style="width:100%;max-width:200px;"
            />
            <span style="font-size:0.875rem;color:#9ca3af;">{pageSizeLabel}</span>
          </label>
        </div>
        <dl class="settings-list">
          <div><dt>Cached sources</dt><dd>{Object.keys(mediaSources).length} media items pre-loaded</dd></div>
        </dl>
      </section>

      <div class="settings-actions">
        <button class="secondary-button" on:click={() => {
          if (confirm("Clear all media cache? This will re-download media on next visit.")) {
            mediaSources = {};
            mediaLoading = {};
            settingsOpen = false;
          }
        }}>Clear media cache</button>
      </div>
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
          <button class="secondary-button" on:click={logoutProfile}>Logout</button>
        {/if}
      </div>
    </div>
  </div>
{/if}
