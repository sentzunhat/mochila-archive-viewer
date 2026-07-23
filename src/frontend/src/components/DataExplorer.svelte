<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { JsonFileRef, JSONPreview } from "../lib/types";
  import { formatter } from "../utils/formatters";

  export let jsonFiles: JsonFileRef[] = [];
  export let selectedJSONIndex = 0;
  export let jsonPreview: JSONPreview | null = null;

  const dispatch = createEventDispatcher<{ selectFile: number }>();
</script>

<section class="messages-layout">
  <aside class="conversation-list">
    {#each jsonFiles as item, index}
      <button class:active={selectedJSONIndex === index} on:click={() => dispatch("selectFile", index)}>
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
