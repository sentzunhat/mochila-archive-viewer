<script lang="ts">
  import type { ArchiveFile, IndexSummary, StructureSection, JsonFileRef } from "../lib/types";
  import { formatter } from "../utils/formatters";

  export let structureSection: StructureSection = "paths";
  export let storePath = "";
  export let providerMediaRoot = "";
  export let providerSnapshotPath = "";
  export let jsonFiles: JsonFileRef[] = [];
  export let selected: ArchiveFile[] = [];
  export let summary: IndexSummary | null = null;
</script>

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
            <p><code>{storePath}</code></p>
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
