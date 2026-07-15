#!/usr/bin/env python3
"""
Surgical patches for remaining features:
- PATCH8b: Add page size range input to settings modal
- PATCH9: Replace loadMore button with paginated counter + infinite scroll sentinel
- PATCH10: Wire up IntersectionObserver in onMount
"""
import subprocess, re, os

SVELTE = "/Users/beltrd/Desktop/projects/sentzunhat/mochila-archive-viewer/src/frontend/src/App.svelte"

with open(SVELTE, 'r') as f:
    content = f.read()

print(f"Read {len(content)} bytes", flush=True)

# ── PATCH 8b: Add page size input in settings modal ──
# Find the setting-row before Theme and insert page size control
settings_search = "        <div class=\"setting-row\">\n          <span>Theme</span>"

if settings_search in content:
    new_settings_block = """        <!-- Page size setting -->
        <div class=\"setting-row\">
          <label for=\"page-size-input\" style=\"display:flex;flex-direction:column;gap:0.25rem;\">
            <span>Items per page (gallery)</span>
            <input 
              id=\"page-size-input\"
              type=\"range\" 
              min=\"30\" 
              max=\"500\" 
              step=\"10\" 
              value={pageSize}
              on:input={(e) => savePageSize(Number((e.target as HTMLInputElement).value))}
              style=\"width:100%;max-width:200px;\"
            />
            <span style=\"font-size:0.875rem;color:#9ca3af;\">{pageSizeLabel}</span>
          </label>
        </div>

        <div class=\"setting-row\">
          <span>Theme</span>"""
    content = content.replace(settings_search, new_settings_block)
    print("✓ PATCH 8b: Added page size slider to settings modal", flush=True)
else:
    # Try alternative whitespace patterns
    alt_patterns = [
        "        <div class='setting-row'>\n          <span>Theme</span>",
        "      <div class=\"setting-row\">\n        <span>Theme</span>",
        "<span>Theme</span>",
    ]
    found = False
    for pat in alt_patterns:
        if pat in content:
            idx = content.index(pat)
            context = content[max(0,idx-50):idx+100]
            print(f"  Found at {idx}: {repr(context[:80])}", flush=True)
            found = True
    if not found:
        print("✗ PATCH 8b: Could not find Theme setting", flush=True)

# ── PATCH 9: Replace loadMore button with paginated counter + sentinel ──
loadmore_search = '<button class="load-more" on:click={loadMore}>Load more</button>'

if loadmore_search in content:
    new_loadmore = """<button class="load-more" on:click={triggerLoadMore} disabled={isLoadingMore}>
                {isLoadingMore ? "Loading..." : `Load more (${paginatedMedia.length} of ${totalMediaCount.toLocaleString()})`}
              </button>
              <!-- Infinite scroll sentinel -->
              <div bind:this={sentinelRef} style="height:1px;"></div>"""
    content = content.replace(loadmore_search, new_loadmore)
    print("✓ PATCH 9: Updated loadMore button + added infinite scroll sentinel", flush=True)
else:
    # Search for similar patterns
    lines = content.split('\n')
    for i, line in enumerate(lines):
        if 'loadMore' in line and ('click' in line or 'on:' in line):
            print(f"  Found loadMore at line {i+1}: {line.strip()}", flush=True)

# ── PATCH 10: Setup observer in onMount and cleanup ──
# Find the end of onMount's try block (before the catch)
onmount_search = '''      // Register keyboard shortcuts for search UX
        document.addEventListener("keydown", handleSearchKeyboard);'''

if onmount_search in content:
    new_onmount_end = '''      // Register keyboard shortcuts for search UX
        document.addEventListener("keydown", handleSearchKeyboard);
        
        // Setup infinite scroll observer after media loads
        setTimeout(() => setupInfiniteScroll(), 500);'''
    content = content.replace(onmount_search, new_onmount_end)
    print("✓ PATCH 10: Added observer setup in onMount", flush=True)
else:
    # Try with different indentation
    alt = 'document.addEventListener("keydown", handleSearchKeyboard);'
    if alt in content:
        idx = content.index(alt)
        context = content[max(0,idx-80):idx+100]
        print(f"  Found at {idx}: {repr(context[:80])}", flush=True)
    else:
        print("✗ PATCH 10: Could not find keyboard shortcut registration", flush=True)

# ── PATCH 11: Add page size reactive label (was PATCH8 partially applied) ──
label_search = '{#if settingsOpen}'
if label_search in content:
    new_label = '''// Page size for gallery display
$: pageSizeLabel = pageSize + " items per page";

{#if settingsOpen}'''
    # Only apply if not already present
    if 'pageSizeLabel' not in content:
        content = content.replace(label_search, new_label)
        print("✓ PATCH 11: Added pageSizeLabel reactive", flush=True)
    else:
        print("ℹ PATCH 11: pageSizeLabel already present", flush=True)

# Write back
with open(SVELTE, 'w') as f:
    f.write(content)

print(f"\nWrote {len(content)} bytes back to {SVELTE}", flush=True)
print("Done! Run 'npm run build' to verify.", flush=True)
