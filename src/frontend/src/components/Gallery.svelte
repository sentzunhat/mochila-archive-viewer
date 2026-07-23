<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { IndexSummary, MediaItem } from "../lib/types";
  import { formatter, mediaName } from "../utils/formatters";
  import { mediaHttpSrc } from "../utils/mediaUrl";

  export let summary: IndexSummary | null;
  export let selectedYear = "all";
  export let visibleMedia: MediaItem[] = [];
  export let totalMediaCount = 0;
  export let paginatedMedia: MediaItem[] = [];
  export let isLoadingMore = false;
  export let hasMoreMedia = true;
  export let activePlatform = "snapchat";
  export let profileId = 0;

  let sentinelRef: HTMLElement | null = null;
  let infiniteObserver: IntersectionObserver | null = null;
  let years: [string, number][] = [];
  let maxYearCount = 0;

  const dispatch = createEventDispatcher<{
    selectMedia: MediaItem;
    changeYear: string;
    loadMore: void;
  }>();

  $: if (summary) {
    years = Object.entries(summary.years).sort(([a], [b]) => b.localeCompare(a));
    maxYearCount = Math.max(...years.map(([, count]) => count), 1);
  }

  $: if (sentinelRef !== undefined) {
    setupInfiniteScroll();
  }

  function setYear(year: string) {
    selectedYear = selectedYear === year ? "all" : year;
    dispatch("changeYear", selectedYear);
  }

  function setupInfiniteScroll() {
    if (infiniteObserver) infiniteObserver.disconnect();
    if (!sentinelRef) return;

    infiniteObserver = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMoreMedia && !isLoadingMore) {
          dispatch("loadMore");
        }
      },
      { rootMargin: "400px" }
    );
    infiniteObserver.observe(sentinelRef);
  }
</script>

<section class="gallery-layout">
  <aside class="year-list" aria-label="Years">
    <button class:active={selectedYear === "all"} on:click={() => setYear("all")}>
      <strong>All</strong>
      <span class="bar"><span class="w-full"></span></span>
      <em>{formatter.format(summary?.mediaCount ?? 0)}</em>
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
            <button class="preview" on:click={() => dispatch("selectMedia", item)} aria-label={`Open ${mediaName(item.entry)}`}>
              {#if item.type === "image"}
                <img loading="lazy" src={mediaHttpSrc(activePlatform, profileId, item.id)} alt={mediaName(item.entry)} />
              {:else}
                <video preload="metadata" controls muted src={mediaHttpSrc(activePlatform, profileId, item.id)} playsinline></video>
              {/if}
            </button>
            <div class="tile-meta">
              <strong title={item.entry}>{mediaName(item.entry)}</strong>
              <span>{item.date} · {item.category} · {item.type}</span>
            </div>
          </article>
        {/each}
      </div>

      {#if hasMoreMedia}
        <button class="load-more" on:click={() => dispatch("loadMore")} disabled={isLoadingMore}>
          {isLoadingMore ? "Loading..." : `Load more (${paginatedMedia.length} of ${totalMediaCount.toLocaleString()})`}
        </button>
        <div bind:this={sentinelRef} class="h-px"></div>
      {/if}
    {/if}
  </div>
</section>

<style>
  .gallery-layout {
    display: grid;
    grid-template-columns: 200px 1fr;
    gap: 1.5rem;
    height: 100%;
    overflow: hidden;
  }

  .year-list {
    border-right: 1px solid var(--accent-soft);
    overflow-y: auto;
    padding: 1rem 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .year-list button {
    background: none;
    border: 1px solid transparent;
    border-radius: 0.375rem;
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    text-align: left;
    font-size: 0.875rem;
    transition: background-color 150ms;
  }

  .year-list button:hover {
    background-color: var(--accent-soft);
  }

  .year-list button.active {
    background-color: var(--accent-dark);
    border-color: var(--accent);
  }

  .year-list strong {
    display: block;
    margin-bottom: 0.25rem;
    font-weight: 600;
  }

  .year-list .bar {
    display: block;
    height: 4px;
    background-color: var(--accent-soft);
    border-radius: 2px;
    margin-bottom: 0.25rem;
    overflow: hidden;
  }

  .year-list .bar span {
    display: block;
    height: 100%;
    background-color: var(--accent);
    transition: width 150ms;
  }

  .year-list em {
    display: block;
    font-size: 0.75rem;
    color: var(--accent-ink);
    font-style: normal;
  }

  .gallery-panel {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    gap: 1rem;
    padding: 1rem;
  }

  .result-line {
    font-size: 0.875rem;
    color: var(--accent-ink);
  }

  .media-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 1rem;
    overflow-y: auto;
    padding-right: 0.5rem;
  }

  .tile {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    min-height: 0;
  }

  .preview {
    background: none;
    border: 1px solid var(--accent-soft);
    border-radius: 0.375rem;
    padding: 0;
    cursor: pointer;
    overflow: hidden;
    aspect-ratio: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: border-color 150ms;
  }

  .preview:hover {
    border-color: var(--accent);
  }

  .preview img,
  .preview video {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .tile-meta {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    min-width: 0;
  }

  .tile-meta strong {
    font-size: 0.75rem;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tile-meta span {
    font-size: 0.625rem;
    color: var(--accent-ink);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .empty {
    padding: 2rem;
    text-align: center;
    color: var(--accent-ink);
  }

  .load-more {
    padding: 0.75rem 1rem;
    background-color: var(--accent-soft);
    border: 1px solid var(--accent);
    border-radius: 0.375rem;
    cursor: pointer;
    font-size: 0.875rem;
    transition: background-color 150ms;
  }

  .load-more:hover:not(:disabled) {
    background-color: var(--accent-dark);
  }

  .load-more:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .h-px {
    height: 1px;
  }
</style>
