<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { IndexSummary, Conversation, UpdateStatus } from "../lib/types";
  import { formatter } from "../utils/formatters";

  export let appVersion: string;
  export let updateStatus: UpdateStatus | null;
  export let summary: IndexSummary | null;
  export let conversations: Conversation[] = [];
  export let storePath: string;
  export let pageSize: number;

  const dispatch = createEventDispatcher<{
    close: void;
    pageSizeChange: number;
    checkUpdate: void;
  }>();

  $: pageSizeLabel = pageSize + " items per page";

  function handlePageSizeInput(event: Event) {
    const target = event.target as HTMLInputElement;
    dispatch("pageSizeChange", Math.max(30, Math.min(500, Number(target.value))));
  }
</script>

<div class="modal-backdrop" role="presentation" on:click={() => dispatch("close")}>
  <div class="settings-modal" role="dialog" aria-modal="true" on:click|stopPropagation on:keydown|stopPropagation>
    <div class="settings-head">
      <div>
        <p class="eyebrow">Settings</p>
        <h2>Application preferences</h2>
      </div>
      <button class="close-button" on:click={() => dispatch("close")} aria-label="Close">×</button>
    </div>

    <section class="settings-section">
      <h3>About</h3>
      <dl class="settings-list">
        <div><dt>Version</dt><dd>Mochila {appVersion}</dd></div>
        <div><dt>Backend</dt><dd>Go with Wails v2</dd></div>
        <div><dt>Storage</dt><dd>{storePath || "~/.mochila/database.sqlite"}</dd></div>
      </dl>
      {#if updateStatus?.available}
        <p class="settings-hint">
          <a href={updateStatus.url} target="_blank" rel="noreferrer">{updateStatus.latest} available — download</a>
        </p>
      {/if}
      <button class="secondary-button" on:click={() => dispatch("checkUpdate")}>Check for updates</button>
    </section>

    <section class="settings-section">
      <h3>Data Management</h3>
      <dl class="settings-list">
        <div><dt>Total media items</dt><dd>{summary ? formatter.format(summary.mediaCount) : "N/A"}</dd></div>
        <div><dt>Archive files indexed</dt><dd>{summary ? summary.zipCount : 0}</dd></div>
        <div><dt>Conversations</dt><dd>{formatter.format(conversations.length)}</dd></div>
      </dl>
      <p class="settings-hint">Indexed data is stored locally in SQLite.</p>
    </section>

    <section class="settings-section">
      <h3>Display</h3>
      <div class="mb-4">
        <label for="page-size-slider" class="flex flex-col gap-1">
          <span>Items per page (gallery)</span>
          <input
            id="page-size-slider"
            type="range"
            min="30"
            max="500"
            step="10"
            value={pageSize}
            on:input={handlePageSizeInput}
            class="w-full max-w-[200px]"
          />
          <span class="text-sm text-archive-muted">{pageSizeLabel}</span>
        </label>
      </div>
    </section>
  </div>
</div>
