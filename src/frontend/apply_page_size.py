#!/usr/bin/env python3
"""Add page size slider to Display settings section."""

SVELTE = "/Users/beltrd/Desktop/projects/sentzunhat/mochila-archive-viewer/src/frontend/src/App.svelte"

with open(SVELTE, 'r') as f:
    content = f.read()

# Target the Display section's batch size display and convert it to a slider
old_display = '''      <section class="settings-section">
        <h3>Display</h3>
        <dl class="settings-list">
          <div><dt>Media batch size</dt><dd>{visibleLimit} items per load</dd>
</div>
          <div><dt>Cached sources</dt><dd>{Object.keys(mediaSources).length} media items pre-loaded</dd></div>
        </dl>
      </section>'''

new_display = '''      <section class="settings-section">
        <h3>Display</h3>
        <div class="setting-row" style="margin-bottom:1rem;">
          <label for="page-size-slider" style="display:flex;flex-direction:column;gap:0.25rem;">
            <span>Items per page (gallery)</span>
            <input 
              id="page-size-slider"
              type="range" 
              min="30" 
              max="500" 
              step="10" 
              value={pageSize}
              on:input={(e) => savePageSize(Number((e.target as HTMLInputElement).value))}
              style="width:100%;max-width:200px;"
            />
            <span style="font-size:0.875rem;color:#9ca3af;">{pageSizeLabel}</span>
          </label>
        </div>
        <dl class="settings-list">
          <div><dt>Cached sources</dt><dd>{Object.keys(mediaSources).length} media items pre-loaded</dd></div>
        </dl>
      </section>'''

if old_display in content:
    content = content.replace(old_display, new_display)
    print("✓ Added page size slider to Display settings")
else:
    # Find what's actually there
    idx = content.find('Media batch size')
    if idx >= 0:
        chunk = content[idx-80:idx+200]
        with open('/tmp/display_section.txt', 'w') as f:
            f.write(chunk)
        print(f"✗ Pattern not found. Actual content around line at {idx}: {repr(chunk[:100])}")
    else:
        print("✗ Could not find Display section at all")

with open(SVELTE, 'w') as f:
    f.write(content)

print(f"Wrote {len(content)} bytes to {SVELTE}")
