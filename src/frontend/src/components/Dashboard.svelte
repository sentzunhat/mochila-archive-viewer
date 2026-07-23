<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { PlatformStatItem } from "../lib/types";
  import { platformThemes } from "../lib/themes";

  export let platformStatsList: PlatformStatItem[] = [];
  export let dashboardNotice = "";

  const dispatch = createEventDispatcher<{
    selectPlatform: string;
    logout: void;
    openExplorer: void;
  }>();
</script>

<main class="min-h-screen bg-archive-bg">
  <div class="max-w-4xl mx-auto p-8">
    <div class="flex justify-between items-center mb-8">
      <div>
        <h1 class="text-3xl font-extrabold text-archive-ink m-0 mb-1">Mochila Archive Viewer</h1>
        <p class="text-archive-muted m-0">Select a platform to explore your archive</p>
      </div>
      <button
        on:click={() => dispatch("logout")}
        class="px-4 py-2 rounded-lg border border-archive-line bg-archive-panel text-archive-ink font-semibold hover:bg-white transition"
      >
        Logout
      </button>
    </div>

    {#if dashboardNotice}
      <p class="mb-6 rounded-lg border border-archive-line bg-archive-panel px-4 py-3 text-sm text-archive-muted">{dashboardNotice}</p>
    {/if}

    {#if platformStatsList.length === 0}
      <section class="empty">
        <h2>No platforms available yet</h2>
        <p>No archive data has been indexed for this profile. Continue to the explorer to add export zips.</p>
        <button class="load-more" on:click={() => dispatch("openExplorer")}>Open archive explorer</button>
      </section>
    {/if}

    <div class="grid gap-6 grid-cols-[repeat(auto-fit,minmax(280px,1fr))]">
      {#each platformStatsList as stat}
        {@const theme = platformThemes[stat.id] ?? platformThemes.snapchat}
        <div
          tabindex="0"
          role="button"
          aria-label="Select {stat.name} platform"
          on:click={() => dispatch("selectPlatform", stat.id)}
          on:keydown={(e) => e.key === "Enter" && dispatch("selectPlatform", stat.id)}
          class="overflow-hidden rounded-2xl border border-archive-line bg-archive-panel cursor-pointer transition shadow-sm hover:shadow-md focus:outline-none focus:ring-2 focus:ring-archive-line {theme.card}"
        >
          <div class="h-1.5 {theme.bar}"></div>
          <div class="p-6">
            <div class="flex justify-between items-start mb-4">
              <h2 class="text-xl font-extrabold text-archive-ink capitalize m-0">{stat.name}</h2>
              <span class="px-3 py-1 rounded-full text-xs font-semibold border {stat.status === 'indexed' ? theme.badge : 'bg-archive-bg text-archive-muted border-archive-line'}">
                {stat.status}
              </span>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <div class="text-2xl font-extrabold text-archive-ink">{stat.mediaCount.toLocaleString()}</div>
                <div class="text-xs text-archive-muted">Media Items</div>
              </div>
              <div>
                <div class="text-2xl font-extrabold text-archive-ink">{stat.zipCount}</div>
                <div class="text-xs text-archive-muted">Zips</div>
              </div>
              <div>
                <div class="text-lg font-bold text-archive-ink">{stat.imageCount.toLocaleString()}</div>
                <div class="text-xs text-archive-muted">Photos</div>
              </div>
              <div>
                <div class="text-lg font-bold text-archive-ink">{stat.videoCount.toLocaleString()}</div>
                <div class="text-xs text-archive-muted">Videos</div>
              </div>
            </div>
            {#if stat.conversationCount > 0}
              <div class="mt-4 pt-3 border-t border-archive-line text-sm text-archive-muted">
                {stat.conversationCount.toLocaleString()} conversations · {stat.jsonFileCount.toLocaleString()} JSON files
              </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  </div>
</main>
