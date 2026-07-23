<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { MediaItem } from "../lib/types";
  import { mediaHttpSrc } from "../utils/mediaUrl";
  import { mediaName } from "../utils/formatters";

  export let item: MediaItem;
  export let activePlatform: string;
  export let profileId: number;

  const dispatch = createEventDispatcher<{ close: void }>();
</script>

<div class="modal-backdrop" role="presentation" on:click={() => dispatch("close")}>
  <div
    class="media-modal"
    role="dialog"
    aria-modal="true"
    aria-label={mediaName(item.entry)}
    tabindex="-1"
    on:click|stopPropagation
    on:keydown={(e) => e.key === "Escape" && dispatch("close")}
  >
    <button class="close-button" on:click={() => dispatch("close")} aria-label="Close">×</button>
    <div class="modal-media">
      {#if item.type === "image"}
        <img src={mediaHttpSrc(activePlatform, profileId, item.id)} alt={mediaName(item.entry)} />
      {:else}
        <video src={mediaHttpSrc(activePlatform, profileId, item.id)} controls autoplay>
          <track kind="captions" />
        </video>
      {/if}
    </div>
    <aside class="modal-meta">
      <p class="eyebrow">Media details</p>
      <h2>{mediaName(item.entry)}</h2>
      <dl>
        <div><dt>Date</dt><dd>{item.date}</dd></div>
        <div><dt>Type</dt><dd>{item.type}</dd></div>
        <div><dt>Category</dt><dd>{item.category}</dd></div>
        <div><dt>Extension</dt><dd>{item.ext}</dd></div>
        <div><dt>Source zip</dt><dd>{item.zip}</dd></div>
        <div><dt>Archive path</dt><dd>{item.entry}</dd></div>
      </dl>
    </aside>
  </div>
</div>
