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
