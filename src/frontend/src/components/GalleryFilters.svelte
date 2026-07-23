<script lang="ts">
  import type { IndexSummary } from "../lib/types";
  import { formatter } from "../utils/formatters";

  export let summary: IndexSummary | null;
  export let selectedYear = "all";
  export let selectedCategory = "all";
  export let selectedType = "all";
  export let searchQuery = "";
  export let storePath = "";
  export let searchInput: HTMLInputElement | undefined;

  let years: [string, number][] = [];
  let categories: [string, number][] = [];
  let types: [string, number][] = [];

  $: if (summary) {
    years = Object.entries(summary.years).sort(([a], [b]) => b.localeCompare(a));
    categories = Object.entries(summary.categories).sort(([a], [b]) => a.localeCompare(b));
    types = Object.entries(summary.types).sort(([a], [b]) => a.localeCompare(b));
  }
</script>

<section class="controls" aria-label="Gallery filters">
  <div class="search-box">
    <input
      type="text"
      placeholder="Search files, categories, dates..."
      bind:value={searchQuery}
      bind:this={searchInput}
      on:input
    />
    {#if searchQuery}
      <button class="clear-search" on:click={() => (searchQuery = "")}>×</button>
    {/if}
  </div>

  <label>
    <span>Year</span>
    <select bind:value={selectedYear} on:change>
      <option value="all">All years</option>
      {#each years as [year, count]}
        <option value={year}>{year} ({formatter.format(count)})</option>
      {/each}
    </select>
  </label>

  <label>
    <span>Category</span>
    <select bind:value={selectedCategory} on:change>
      <option value="all">All categories</option>
      {#each categories as [category, count]}
        <option value={category}>{category} ({formatter.format(count)})</option>
      {/each}
    </select>
  </label>

  <label>
    <span>Type</span>
    <select bind:value={selectedType} on:change>
      <option value="all">All types</option>
      {#each types as [type, count]}
        <option value={type}>{type} ({formatter.format(count)})</option>
      {/each}
    </select>
  </label>

  <label>
    <span>Storage</span>
    <input value={storePath} readonly />
  </label>
</section>
