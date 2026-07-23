<script lang="ts">
  import type { IndexSummary, Conversation, ArchiveFile } from "../lib/types";
  import { formatter } from "../utils/formatters";

  export let summary: IndexSummary;
  export let conversations: Conversation[] = [];
  export let selected: ArchiveFile[] = [];
</script>

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
