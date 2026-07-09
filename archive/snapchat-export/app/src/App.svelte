<script lang="ts">
  import type { ArchiveSummary, ConversationSummary, JsonSummary, MediaItem } from "./types";

  type View = "gallery" | "today" | "messages" | "data" | "structure";

  let summary: ArchiveSummary | null = null;
  let media: MediaItem[] = [];
  let conversations: ConversationSummary[] = [];
  let view: View = "gallery";
  let search = "";
  let selectedYear = "all";
  let selectedCategory = "all";
  let selectedType = "all";
  let visibleLimit = 180;
  let selectedJsonIndex: number | null = null;
  let selectedJson: JsonSummary | null = null;
  let selectedConversationId: string | null = null;
  let selectedMedia: MediaItem | null = null;
  let loadingJson = false;
  let error: string | null = null;

  const formatter = new Intl.NumberFormat();

  $: years = summary
    ? Object.entries(summary.years).sort(([a], [b]) => b.localeCompare(a))
    : [];
  $: categories = summary
    ? Object.entries(summary.categories).sort(([a], [b]) => a.localeCompare(b))
    : [];
  $: types = summary ? Object.entries(summary.types) : [];
  $: maxYearCount = Math.max(...years.map(([, count]) => count), 1);
  $: filteredMedia = media.filter((item) => {
    const query = search.trim().toLowerCase();

    if (selectedYear !== "all" && item.year !== selectedYear) return false;
    if (selectedCategory !== "all" && item.category !== selectedCategory) return false;
    if (selectedType !== "all" && item.type !== selectedType) return false;
    if (!query) return true;

    return `${item.entry} ${item.date} ${item.category} ${item.type} ${item.zip}`
      .toLowerCase()
      .includes(query);
  });
  $: visibleMedia = filteredMedia.slice(0, visibleLimit);
  $: todayKey = new Date().toISOString().slice(5, 10);
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

  function setYear(year: string) {
    selectedYear = selectedYear === year ? "all" : year;
    visibleLimit = 180;
  }

  function keepModalOpen(event: MouseEvent) {
    event.stopPropagation();
  }

  function handleModalKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      selectedMedia = null;
      return;
    }

    event.stopPropagation();
  }

  async function loadJson(index: number) {
    selectedJsonIndex = index;
    loadingJson = true;
    selectedJson = null;

    try {
      const response = await fetch(`/api/json?index=${index}`);
      if (!response.ok) throw new Error(`JSON request failed: ${response.status}`);
      selectedJson = await response.json();
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    } finally {
      loadingJson = false;
    }
  }

  async function loadArchive() {
    try {
      const [summaryResponse, mediaResponse, messagesResponse] = await Promise.all([
        fetch("/api/summary"),
        fetch("/api/media"),
        fetch("/api/messages"),
      ]);

      if (!summaryResponse.ok) throw new Error(`Summary request failed: ${summaryResponse.status}`);
      if (!mediaResponse.ok) throw new Error(`Media request failed: ${mediaResponse.status}`);
      if (!messagesResponse.ok) throw new Error(`Messages request failed: ${messagesResponse.status}`);

      summary = await summaryResponse.json();
      media = await mediaResponse.json();
      conversations = await messagesResponse.json();
      selectedConversationId = conversations[0]?.id ?? null;
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);
    }
  }

  loadArchive();
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
{:else if !summary}
  <main class="center">
    <section class="empty">
      <h1>Indexing Snapchat export...</h1>
      <p>The app is reading local zip indexes only. Your raw files stay in `tools/snapchat-export/inbox/`.</p>
    </section>
  </main>
{:else}
  <header>
    <div class="topbar">
      <div>
        <p class="eyebrow">Snapchat export loaded</p>
        <h1>Mochila</h1>
        <p class="subtitle">
          {summary.zipCount} zip files · {humanBytes(summary.totalZipBytes)} ·
          {formatter.format(summary.totalEntries)} archive entries
        </p>
      </div>

      <nav class="tabs" aria-label="Archive views">
        <button class:active={view === "gallery"} on:click={() => (view = "gallery")}>Gallery</button>
        <button class:active={view === "today"} on:click={() => (view = "today")}>On This Day</button>
        <button class:active={view === "messages"} on:click={() => (view = "messages")}>Messages</button>
        <button class:active={view === "data"} on:click={() => (view = "data")}>Data Explorer</button>
        <button class:active={view === "structure"} on:click={() => (view = "structure")}>Structure</button>
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
        <strong>{formatter.format(summary.jsonCount)}</strong>
        <span>JSON files</span>
      </article>
      <article>
        <strong>{humanBytes(summary.totalZipBytes)}</strong>
        <span>private local data</span>
      </article>
    </section>

    {#if view === "gallery"}
      <section class="controls" aria-label="Gallery filters">
        <label>
          <span>Search</span>
          <input
            bind:value={search}
            on:input={() => (visibleLimit = 180)}
            placeholder="Search dates, categories, filenames..."
          />
        </label>
        <label>
          <span>Year</span>
          <select bind:value={selectedYear} on:change={() => (visibleLimit = 180)}>
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
      </section>

      <section class="gallery-layout">
        <aside class="year-list" aria-label="Years">
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
                      <img loading="lazy" src={`/media/${item.id}`} alt="" />
                    {:else}
                      <video preload="metadata" controls muted src={`/media/${item.id}`}></video>
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
              <button class="load-more" on:click={() => (visibleLimit += 180)}>
                Load more
              </button>
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
                    <img loading="lazy" src={`/media/${item.id}`} alt="" />
                  {:else}
                    <video preload="metadata" controls muted src={`/media/${item.id}`}></video>
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
              on:click={() => (selectedConversationId = conversation.id)}
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
          {#if !selectedConversation}
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
              {#each selectedConversation.messages as message}
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
    {:else if view === "data"}
      <section class="data-layout">
        <aside class="json-list">
          {#each summary.jsonFiles as file, index}
            <button class:active={selectedJsonIndex === index} on:click={() => loadJson(index)}>
              <strong>{file.entry}</strong>
              <span>{file.zip}</span>
            </button>
          {/each}
        </aside>

        <section class="json-panel">
          {#if loadingJson}
            <div class="empty">Reading JSON from the zip...</div>
          {:else if selectedJson}
            <div class="json-heading">
              <div>
                <h2>{selectedJson.file.entry}</h2>
                <p>{selectedJson.summary.kind} · {formatter.format(selectedJson.summary.records)} records/keys</p>
              </div>
            </div>

            {#if selectedJson.summary.childCounts}
              <div class="child-grid">
                {#each selectedJson.summary.childCounts as child}
                  <article>
                    <strong>{child.key}</strong>
                    <span>{child.type}{child.records === undefined ? "" : ` · ${formatter.format(child.records)}`}</span>
                  </article>
                {/each}
              </div>
            {/if}

            <pre>{JSON.stringify(selectedJson.summary.sample, null, 2)}</pre>
          {:else}
            <div class="empty">Choose a JSON file to inspect its shape and sample data.</div>
          {/if}
        </section>
      </section>
    {:else}
      <section class="structure">
        <div class="zip-grid">
          {#each summary.zips as zip}
            <article>
              <strong>{zip.name}</strong>
              <span>{humanBytes(zip.size)} · {formatter.format(zip.entries)} entries</span>
            </article>
          {/each}
        </div>
        <pre>{JSON.stringify(summary, null, 2)}</pre>
      </section>
    {/if}
  </main>

  {#if selectedMedia}
    <div class="modal-backdrop" role="presentation" on:click={() => (selectedMedia = null)}>
      <div
        class="media-modal"
        role="dialog"
        aria-modal="true"
        aria-label={mediaName(selectedMedia)}
        tabindex="-1"
        on:click={keepModalOpen}
        on:keydown={handleModalKeydown}
      >
        <button class="close-button" on:click={() => (selectedMedia = null)} aria-label="Close">×</button>
        <div class="modal-media">
          {#if selectedMedia.type === "image"}
            <img src={`/media/${selectedMedia.id}`} alt="" />
          {:else}
            <video src={`/media/${selectedMedia.id}`} controls autoplay>
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
