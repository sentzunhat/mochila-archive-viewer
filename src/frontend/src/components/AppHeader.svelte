<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { IndexSummary, View, Theme, Profile } from "../lib/types";
  import { formatter } from "../utils/formatters";

  export let summary: IndexSummary;
  export let activeTheme: Theme;
  export let storePath: string;
  export let view: View;
  export let selecting = false;
  export let profile: Profile;

  const dispatch = createEventDispatcher<{
    changeView: View;
    pickZips: void;
    openSettings: void;
    openProfile: void;
    goToDashboard: void;
  }>();
</script>

<header>
  <div class="topbar">
    <div>
      <p class="eyebrow flex items-center gap-2">
        <span class="inline-block h-2.5 w-2.5 rounded-full border border-archive-line {activeTheme.bar}"></span>
        {activeTheme.name} export loaded
      </p>
      <h1>Mochila</h1>
      <p class="subtitle">
        {summary.zipCount} zip files · {formatter.format(summary.mediaCount)} media items · stored in <code>{storePath}</code>
      </p>
    </div>

    <nav class="tabs" aria-label="Archive views">
      <button class:active={view === "gallery"} on:click={() => dispatch("changeView", "gallery")}>Gallery</button>
      <button class:active={view === "today"} on:click={() => dispatch("changeView", "today")}>On This Day</button>
      <button class:active={view === "messages"} on:click={() => dispatch("changeView", "messages")}>Messages</button>
      <button class:active={view === "structure"} on:click={() => dispatch("changeView", "structure")}>Structure</button>
      <button class:active={view === "data"} on:click={() => dispatch("changeView", "data")}>Data Explorer</button>
      <button on:click={() => dispatch("goToDashboard")}>← Dashboard</button>
      <button on:click={() => dispatch("pickZips")} disabled={selecting}>
        {selecting ? "Opening..." : "Add zips"}
      </button>
      <button on:click={() => dispatch("openSettings")}>⚙ Settings</button>
      <button on:click={() => dispatch("openProfile")}>
        {profile.loggedIn ? (profile.fullName || profile.username) : "Profile"}
      </button>
    </nav>
  </div>
</header>
