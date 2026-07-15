#!/usr/bin/env python3
"""Fix page size slider in Display settings."""

SVELTE = "/Users/beltrd/Desktop/projects/sentzunhat/mochila-archive-viewer/src/frontend/src/App.svelte"

with open(SVELTE, 'r') as f:
    content = f.read()

# The actual pattern from the file output - use exact text between Display and Data Management
old_display_block = """<h3>Display</h3>
        <dl class="settings-list">
          <div><dt>Media batch size</dt><dd>{visibleLimit} items per load</dd></div>
          <div><dt>Cached sources</dt><dd>{Object.keys(mediaSources).length} media items pre-loaded</dd></div>
        </dl>"""

new_display_block = """<h3>Display</h3>
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
        </dl>"""

if old_display_block in content:
    content = content.replace(old_display_block, new_display_block)
    print("✓ Applied page size slider patch")
else:
    # Try to find and write actual pattern
    idx = content.find('<h3>Display</h3>')
    if idx >= 0:
        actual = content[idx:idx+280]
        with open(SVELTE + '.tmp', 'w') as f:
            f.write(actual)
        print(f"✗ Not found. Actual text:")
        for i, line in enumerate(actual.split('\n')):
            print(f"  {i}: {repr(line)}")

with open(SVELTE, 'w') as f:
    f.write(content)
