<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { MediaItem } from "../lib/types";
  import { formatter, mediaName } from "../utils/formatters";
  import { mediaHttpSrc } from "../utils/mediaUrl";

  export let onThisDayMedia: MediaItem[] = [];
  export let todayKey = "";
  export let activePlatform = "snapchat";
  export let profileId = 0;

  const dispatch = createEventDispatcher<{ selectMedia: MediaItem }>();
</script>

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
  {/if}
</section>
