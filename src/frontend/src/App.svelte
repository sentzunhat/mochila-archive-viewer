<script lang="ts">
  import { onMount } from "svelte";
  import {
    GetFrontendState,
    GetPlatformSnapshot,
    GetMedia,
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
  type Profile = { username: string; fullName: string; loggedIn: boolean };
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
  let mediaSources: Record<number, string> = {};
  let mediaLoading: Record<number, "pending" | "resolved" | "failed"> = {};
  let profileOpen = false;
  let profileUsername = "";
  let profileFullName = "";
  let structureSection: StructureSection = "paths";

  const formatter = new Intl.NumberFormat();
  $: storeRoot = appState.storePath ? appState.storePath.replace("/database.sqlite", "") : "~/.mochila";
  $: providerRoot = `${storeRoot}/indexed/providers/${activePlatform}`;
  $: providerMediaRoot = `${providerRoot}/media`;
  $: providerSnapshotPath = `${providerRoot}/snapshot.json`;
  $: localDate = new Date();
  $: todayKey = `${String(localDate.getMonth() + 1).padStart(2, "0")}-${String(localDate.getDate()).padStart(2, "0")}`;

  $: years = summary ? Object.entries(summary.years).sort(([a], [b]) => b.localeCompare(a)) : [];
  $: categories = summary ? Object.entries(summary.categories).sort(([a], [b]) => a.localeCompare(b)) : [];
  $: types = summary ? Object.entries(summary.types).sort(([a], [b]) => a.localeCompare(b)) : [];
  $: filteredMedia = media.filter((item) => {
    if (selectedYear !== "all" && item.year !== selectedYear) return false;
    if (selectedCategory !== "all" && item.category !== selectedCategory) return false;
    if (selectedType !== "all" && item.type !== selectedType) return false;
    return true;
  });
  $: visibleMedia = filteredMedia.slice(0, visibleLimit);
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
      return true;
    }).slice(0, 96);
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
    await ensureMediaSources(media.slice(0, 48));
  }

  onMount(async () => {
    try {
      appState = await GetFrontendState();
      profileUsername = appState.profile.username ?? "";
      profileFullName = appState.profile.fullName ?? "";
      await loadPlatform(activePlatform);
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    } finally {
      loading = false;
    }
  });

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
      profileOpen = false;
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    }
  }

  async function logoutProfile() {
    try {
      const profile = await LogoutProfile();
      appState = { ...appState, profile };
      profileUsername = "";
      profileFullName = "";
      profileOpen = false;
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
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
          <select bind:value={selectedCategory}>
            <option value="all">All categories</option>
            {#each categories as [category, count]}
              <option value={category}>{category} ({formatter.format(count)})</option>
            {/each}
          </select>
        </label>
        <label>
          <span>Type</span>
          <select bind:value={selectedType}>
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
              <span class="bar"><span style={`width:${Math.max(4, (count / summary.mediaCount) * 100)}%`}></span></span>
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
                      <video preload="metadata" muted src={sourceFor(item)}></video>
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
              <button class="load-more" on:click={loadMore}>Load more</button>
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
                    <video preload="metadata" muted src={sourceFor(item)}></video>
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
      <div class="media-modal" role="dialog" aria-modal="true" aria-label={mediaName(selectedMedia)} tabindex="-1">
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

{#if profileOpen}
  <div class="modal-backdrop" role="presentation" on:click={() => (profileOpen = false)}>
    <div class="profile-modal" role="dialog" aria-modal="true" on:click|stopPropagation on:keydown|stopPropagation>
      <div class="profile-head">
        <div>
          <p class="eyebrow">Profile</p>
          <h2>{appState.profile.loggedIn ? "Your archive profile" : "Create a simple profile"}</h2>
        </div>
        <button class="close-button" on:click={() => (profileOpen = false)} aria-label="Close">×</button>
      </div>
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
          <button class="secondary-button" on:click={logoutProfile}>Logout</button>
        {/if}
      </div>
    </div>
  </div>
{/if}
