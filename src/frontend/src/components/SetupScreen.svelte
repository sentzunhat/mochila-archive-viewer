<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { ArchiveFile } from "../lib/types";

  export let platformName: string;
  export let storePath: string;
  export let selected: ArchiveFile[] = [];
  export let selecting = false;
  export let indexing = false;

  const dispatch = createEventDispatcher<{ pickZips: void; indexSelected: void }>();
</script>

<main class="center">
  <section class="empty">
    <h1>Indexing {platformName} export...</h1>
    <p>Your selected zips and indexed cache live in <code>{storePath || "~/.mochila/database.sqlite"}</code>.</p>
    <p>
      {#if selected.length === 0}
        <button class="load-more" on:click={() => dispatch("pickZips")} disabled={selecting}>
          {selecting ? "Opening picker..." : "Choose export zip files"}
        </button>
      {:else}
        <button class="load-more" on:click={() => dispatch("indexSelected")} disabled={indexing}>
          {indexing ? "Indexing selected archives..." : `Index ${selected.length} zip${selected.length === 1 ? "" : "s"}`}
        </button>
      {/if}
    </p>
  </section>
</main>
