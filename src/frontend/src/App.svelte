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
  import type {
    FrontendState,
    Profile,
    ArchiveFile,
    IndexSummary,
    MediaItem,
    Conversation,
    JsonFileRef,
    JSONPreview,
    PlatformStatItem,
    View,
    StructureSection,
    UpdateStatus,
  } from "./lib/types";
  import { platformThemes } from "./lib/themes";
  import { formatter, mediaName } from "./utils/formatters";
  import { mediaHttpSrc } from "./utils/mediaUrl";
  import LoginScreen from "./components/LoginScreen.svelte";
  import Dashboard from "./components/Dashboard.svelte";
  import SetupScreen from "./components/SetupScreen.svelte";
  import AppHeader from "./components/AppHeader.svelte";
  import StatBar from "./components/StatBar.svelte";
  import MediaModal from "./components/MediaModal.svelte";
  import SettingsModal from "./components/SettingsModal.svelte";
  import ProfileModal from "./components/ProfileModal.svelte";
  import GalleryFilters from "./components/GalleryFilters.svelte";
  import Gallery from "./components/Gallery.svelte";
  import OnThisDay from "./components/OnThisDay.svelte";
  import Messages from "./components/Messages.svelte";
  import Structure from "./components/Structure.svelte";
  import DataExplorer from "./components/DataExplorer.svelte";

  // ── Global state ──
  let appState: FrontendState = {
    name: "Mochila",
    tagline: "",
    providers: [],
    storePath: "",
    profile: { id: 0, username: "", fullName: "", loggedIn: false },
  };
  let loading = true;
  let error: string | null = null;

  // ── Login/Dashboard flow ──
  let showLoginScreen = false;
  let showDashboard = false;
  let platformStatsList: PlatformStatItem[] = [];
  let dashboardNotice = "";
  let authError = "";

  // ── Platform state ──
  let activePlatform = "snapchat";
  let selectedPlatform: string | null = null;
  let selected: ArchiveFile[] = [];
  let summary: IndexSummary | null = null;
  let media: MediaItem[] = [];
  let jsonFiles: JsonFileRef[] = [];
  let conversations: Conversation[] = [];

  // ── UI state ──
  let view: View = "gallery";
  let selectedYear = "all";
  let selectedCategory = "all";
  let selectedType = "all";
  let searchQuery = "";
  let selectedConversationId: string | null = null;
  let selectedJSONIndex = 0;
  let jsonPreview: JSONPreview | null = null;
  let selectedMedia: MediaItem | null = null;
  let structureSection: StructureSection = "paths";
  let profileOpen = false;
  let settingsOpen = false;

  // ── Profile state ──
  let profileUsername = "";
  let profileFullName = "";
  let availableUsers: { id: number; username: string; fullName: string }[] = [];

  // ── Gallery pagination ──
  let paginatedMedia: MediaItem[] = [];
  let totalMediaCount = 0;
  let pageSize = 180;
  let currentOffset = 0;
  let isLoadingMore = false;
  let hasMoreMedia = true;
  let selecting = false;
  let indexing = false;
  let loadingConversation = false;
  let searchInput: HTMLInputElement | undefined;

  // ── Derived state ──
  let loadedConversationIds = new Set<string>();
  let messageMediaCache: Record<number, MediaItem> = {};
  let mediaGeneration = 0;

  $: activeTheme = platformThemes[activePlatform] ?? platformThemes.snapchat;
  $: storeRoot = appState.storePath ? appState.storePath.replace("/database.sqlite", "") : "~/.mochila";
  $: providerRoot = `${storeRoot}/indexed/providers/${activePlatform}`;
  $: providerMediaRoot = `${providerRoot}/media`;
  $: providerSnapshotPath = `${providerRoot}/snapshot.json`;
  $: localDate = new Date();
  $: todayKey = `${String(localDate.getMonth() + 1).padStart(2, "0")}-${String(
    localDate.getDate()
  ).padStart(2, "0")}`;
  $: onThisDayMedia = media.filter((item) => item.date.slice(5, 10) === todayKey);
  $: selectedConversation = conversations.find((c) => c.id === selectedConversationId) ?? conversations[0] ?? null;

  // ── Theme CSS ──
  $: if (typeof document !== "undefined" && activeTheme) {
    const s = document.documentElement.style;
    s.setProperty("--accent", activeTheme.accent);
    s.setProperty("--accent-dark", activeTheme.dark);
    s.setProperty("--accent-soft", activeTheme.soft);
    s.setProperty("--accent-ink", activeTheme.ink);
  }

  // ── Auto-load conversation messages ──
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

  // ── Search keyboard shortcuts ──
  function handleSearchKeyboard(event: KeyboardEvent) {
    const target = event.target as HTMLElement;
    const tag = target?.tagName;
    if (tag === "INPUT" && target !== searchInput) return;
    if (tag === "TEXTAREA" || tag === "SELECT") return;

    if (event.key === "/" && !event.metaKey && !event.ctrlKey && !event.altKey) {
      event.preventDefault();
      searchInput?.focus();
      return;
    }

    if (event.key === "Escape" && document.activeElement === searchInput) {
      event.preventDefault();
      searchQuery = "";
      searchInput?.blur();
      return;
    }

    if ((event.metaKey || event.ctrlKey) && event.key === "f") {
      event.preventDefault();
      searchInput?.focus();
      searchInput?.select();
    }
  }

  // ── Login handlers ──
  async function handleLogin(e: CustomEvent<{ username: string; fullname: string }>) {
    const { username, fullname } = e.detail;
    if (!username.trim()) {
      authError = "Username is required";
      return;
    }
    try {
      await SaveProfile(username.trim(), fullname.trim() || username.trim());
      showLoginScreen = false;
      showDashboard = true;
      await loadPlatformDashboard();
    } catch (err) {
      authError = "Login failed: " + String(err);
    }
  }

  async function handleLogout() {
    try {
      const nextProfile = await LogoutProfile();
      appState = { ...appState, profile: nextProfile };
    } catch {}
    showLoginScreen = true;
    showDashboard = false;
    profileOpen = false;
    profileUsername = "";
    profileFullName = "";
    error = null;
    authError = "";
    dashboardNotice = "";
    paginatedMedia = [];
    selectedPlatform = null;
    currentOffset = 0;
    hasMoreMedia = true;
    loadedConversationIds.clear();
  }

  // ── Dashboard ──
  async function loadPlatformDashboard() {
    try {
      const nextStats: PlatformStatItem[] = [];
      const platforms = ["snapchat", "instagram", "facebook"];
      for (const p of platforms) {
        let stats: any;
        try {
          stats = await GetPlatformStats(p);
        } catch {
          continue;
        }
        if (stats) {
          nextStats.push({
            id: p,
            name: p.charAt(0).toUpperCase() + p.slice(1),
            status: stats.mediaCount > 0 ? "indexed" : "empty",
            mediaCount: Number(stats.mediaCount || 0),
            imageCount: Number(stats.imageCount || 0),
            videoCount: Number(stats.videoCount || 0),
            zipCount: Number(stats.zipCount || 0),
            conversationCount: Number(stats.conversationCount || 0),
            jsonFileCount: Number(stats.jsonFileCount || 0),
            yearsFound: Number(stats.yearsFound || 0),
          });
        }
      }
      platformStatsList = nextStats;
    } catch (e) {
      console.error("Failed to load dashboard stats:", e);
    }
  }

  async function selectPlatform(pid: string) {
    dashboardNotice = "";
    const prevPlatform = activePlatform;
    selectedPlatform = pid;
    showDashboard = false;
    paginatedMedia = [];
    totalMediaCount = 0;
    currentOffset = 0;
    hasMoreMedia = true;
    try {
      await loadPlatform(pid);
      await loadMediaBatch();
    } catch (e) {
      activePlatform = prevPlatform;
      selectedPlatform = null;
      showDashboard = true;
      error = null;
      const name = platformThemes[pid]?.name ?? pid;
      dashboardNotice = `${name} isn't supported yet — Snapchat is the only live provider for now.`;
    }
  }

  // ── Platform loading ──
  async function loadPlatform(pid: string) {
    activePlatform = pid;
    error = null;

    const snapshot = await GetPlatformSnapshot(pid);
    selected = snapshot?.selected ?? [];
    summary = snapshot?.summary ?? null;
    media = snapshot?.media ?? [];
    jsonFiles = snapshot?.jsonFiles ?? [];
    conversations = snapshot?.conversations ?? [];
    selectedConversationId = conversations[0]?.id ?? null;
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

  // ── Archive selection ──
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
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
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

      if (showDashboard && selectedPlatform === activePlatform) {
        await loadPlatformDashboard();
      }
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      indexing = false;
    }
  }

  // ── Media filtering ──
  function currentMediaFilter() {
    return {
      Year: selectedYear,
      Category: selectedCategory,
      Type: selectedType,
      Search: searchQuery.trim(),
    };
  }

  let filterReloadTimer: ReturnType<typeof setTimeout> | undefined;
  function reloadFilteredMedia(immediate = false) {
    if (!selectedPlatform) return;
    const run = () => {
      mediaGeneration += 1;
      paginatedMedia = [];
      currentOffset = 0;
      hasMoreMedia = true;
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

  async function loadMediaBatch() {
    if (isLoadingMore || !hasMoreMedia || !selectedPlatform) return;
    const gen = mediaGeneration;
    isLoadingMore = true;

    try {
      const platform = selectedPlatform!;
      const filter = currentMediaFilter();

      if (currentOffset === 0) {
        try {
          const count = await GetMediaCount(platform, filter);
          if (gen === mediaGeneration) totalMediaCount = count;
        } catch {}
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

  // ── Conversations ──
  async function openConversation(id: string) {
    loadingConversation = true;
    error = null;
    try {
      selectedConversationId = id;
      const full = await GetConversation(activePlatform, id);
      loadedConversationIds.add(id);
      if (full) {
        conversations = conversations.map((c) => (c.id === id ? full : c));
      }
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loadingConversation = false;
    }
  }

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
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  // ── JSON Explorer ──
  async function openJSONFile(index: number) {
    error = null;
    selectedJSONIndex = index;
    try {
      jsonPreview = await GetJSONPreview(activePlatform, index);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  // ── Profile Management ──
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
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  async function switchUser(userId: number) {
    try {
      const profile = await SelectUser(userId);
      appState = { ...appState, profile };
      profileUsername = profile.username ?? "";
      profileFullName = profile.fullName ?? "";
      profileOpen = false;
      loadedConversationIds.clear();
      await loadPlatform(activePlatform);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  async function loadAvailableUsers() {
    try {
      availableUsers = await AvailableUsers();
    } catch {}
  }

  // ── Settings ──
  async function savePageSize(newSize: number) {
    pageSize = Math.max(30, Math.min(500, newSize));
    try {
      await SaveAppSettings({ pagesize: pageSize, loggedin: false });
    } catch {}
  }

  let appVersion = "dev";
  let updateStatus: UpdateStatus | null = null;

  async function checkForUpdate() {
    try {
      updateStatus = await CheckForUpdate();
    } catch {}
  }

  // ── Lifecycle ──
  onMount(async () => {
    try {
      appState = await GetFrontendState();
      profileUsername = appState.profile.username ?? "";
      profileFullName = appState.profile.fullName ?? "";
      try {
        appVersion = await AppVersion();
      } catch {}
      try {
        updateStatus = await CheckForUpdate();
      } catch {}

      const settings = await GetAppSettings();
      if (settings) pageSize = Number(settings.pagesize) || 180;

      if (!appState.profile.loggedIn) {
        showLoginScreen = true;
      } else {
        showDashboard = true;
        await loadPlatformDashboard();
      }
      await loadAvailableUsers();
      await loadPlatform(activePlatform);

      document.addEventListener("keydown", handleSearchKeyboard);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  });
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
  <LoginScreen {authError} on:login={handleLogin} />
{:else if showDashboard}
  <Dashboard
    {platformStatsList}
    {dashboardNotice}
    on:selectPlatform={(e) => selectPlatform(e.detail)}
    on:logout={handleLogout}
    on:openExplorer={() => (showDashboard = false)}
  />
{:else if !summary}
  <SetupScreen
    platformName={activeTheme.name}
    storePath={appState.storePath}
    {selected}
    {selecting}
    {indexing}
    on:pickZips={pickZips}
    on:indexSelected={indexSelected}
  />
{:else}
  <AppHeader
    {summary}
    {activeTheme}
    storePath={appState.storePath}
    {view}
    {selecting}
    profile={appState.profile}
    on:changeView={(e) => (view = e.detail)}
    on:pickZips={pickZips}
    on:openSettings={() => (settingsOpen = true)}
    on:openProfile={() => (profileOpen = true)}
    on:goToDashboard={async () => {
      selectedPlatform = null;
      showDashboard = true;
      await loadPlatformDashboard();
    }}
  />

  <main>
    <StatBar {summary} {conversations} {selected} />

    {#if view === "gallery"}
      <GalleryFilters
        {summary}
        bind:selectedYear
        bind:selectedCategory
        bind:selectedType
        bind:searchQuery
        storePath={appState.storePath}
        bind:searchInput
        on:input={() => reloadFilteredMedia()}
      />
      <Gallery
        {summary}
        {selectedYear}
        visibleMedia={paginatedMedia}
        {totalMediaCount}
        {paginatedMedia}
        {isLoadingMore}
        {hasMoreMedia}
        {activePlatform}
        profileId={appState.profile.id}
        on:selectMedia={(e) => (selectedMedia = e.detail)}
        on:changeYear={(e) => {
          selectedYear = e.detail;
          reloadFilteredMedia(true);
        }}
        on:loadMore={loadMediaBatch}
      />
    {:else if view === "today"}
      <OnThisDay
        {onThisDayMedia}
        {todayKey}
        {activePlatform}
        profileId={appState.profile.id}
        on:selectMedia={(e) => (selectedMedia = e.detail)}
      />
    {:else if view === "messages"}
      <Messages
        {conversations}
        {selectedConversation}
        {loadingConversation}
        {activePlatform}
        profileId={appState.profile.id}
        on:selectConversation={(e) => openConversation(e.detail)}
        on:selectMessageMedia={(e) => openMessageMedia(e.detail)}
      />
    {:else if view === "structure"}
      <Structure
        bind:structureSection
        storePath={appState.storePath}
        {providerMediaRoot}
        {providerSnapshotPath}
        {jsonFiles}
        {selected}
        {summary}
      />
    {:else}
      <DataExplorer
        {jsonFiles}
        bind:selectedJSONIndex
        {jsonPreview}
        on:selectFile={(e) => openJSONFile(e.detail)}
      />
    {/if}
  </main>

  {#if selectedMedia}
    <MediaModal
      item={selectedMedia}
      {activePlatform}
      profileId={appState.profile.id}
      on:close={() => (selectedMedia = null)}
    />
  {/if}
{/if}

{#if settingsOpen}
  <SettingsModal
    {appVersion}
    {updateStatus}
    {summary}
    {conversations}
    storePath={appState.storePath}
    {pageSize}
    on:close={() => (settingsOpen = false)}
    on:pageSizeChange={(e) => savePageSize(e.detail)}
    on:checkUpdate={checkForUpdate}
  />
{/if}

{#if profileOpen}
  <ProfileModal
    profile={appState.profile}
    {availableUsers}
    bind:profileUsername
    bind:profileFullName
    on:close={() => (profileOpen = false)}
    on:save={(e) => {
      profileUsername = e.detail.username;
      profileFullName = e.detail.fullName;
      saveProfileForm();
    }}
    on:switchUser={(e) => switchUser(e.detail)}
    on:newProfile={() => {
      profileUsername = "";
      profileFullName = "";
    }}
    on:logout={handleLogout}
  />
{/if}
