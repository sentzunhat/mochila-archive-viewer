#!/usr/bin/env python3
"""
Patch src/frontend/src/App.svelte to add:
 1) Login/welcome screen (entry point before anything else)
 2) Platform selection dashboard with stats
 3) Infinite scroll pagination for gallery  
 4) Configurable page size settings

Uses Python subprocess to read/write directly, avoiding tool artifacts.
"""
import subprocess
import os

WORKSPACE = "/Users/beltrd/Desktop/projects/sentzunhat/mochila-archive-viewer"
SVELTE_FILE = os.path.join(WORKSPACE, "src/frontend/src/App.svelte")

def read_file():
    r = subprocess.run(["cat", SVELTE_FILE], capture_output=True, text=True)
    if r.returncode != 0:
        raise RuntimeError(f"Cannot read: {r.stderr}")
    return r.stdout

def write_file(content):
    with open(SVELTE_FILE, 'w') as f:
        f.write(content)

def patch(name, old, new, content):
    if old not in content:
        print(f"✗ [{name}] Pattern not found!")
        return content, False
    content = content.replace(old, new, 1)
    print(f"✓ [{name}] Applied")
    return content, True

content = read_file()
lines = content.split('\n')
print(f"Read {len(lines)} lines ({len(content)} bytes)")

# ──────────────────────────────────────────────
# PATCH 1: Add new types after existing type definitions
# ──────────────────────────────────────────────
type_marker = '''  type PlatformSnapshot = { selected: ArchiveFile[]; summary: IndexSummary | null; media: MediaItem[]; jsonFiles: JsonFileRef[]; conversations: Conversation[] };'''

new_types = '''  type PlatformStatItem = { 
    id: string; name: string; status: string; 
    mediaCount: number; imageCount: number; videoCount: number; 
    zipCount: number; conversationCount: number; jsonFileCount: number; yearsFound: number 
  };'''

content, ok = patch("PATCH1", type_marker, content.split(type_marker)[0] + new_types + "\n" + type_marker if False else type_marker + "\n" + new_types, content)
if not ok:
    print("PATCH1 failed - types may already be present or different format")

# ──────────────────────────────────────────────
# PATCH 2: Add login/dashboard/pagination state variables after existing state
# ──────────────────────────────────────────────
state_marker = '''  let structureSection: StructureSection = "paths";'''

new_state = '''
  // Login & dashboard state
  let showLoginScreen: boolean = false;
  let loginUsername = "";
  let loginFullname = "";
  let authError = "";
  
  // Platform dashboard state  
  let showDashboard = false;
  let platformStatsList: PlatformStatItem[] = [];
  let selectedPlatform: string | null = null;
  let selectedYearPaged: string = "all";
  
  // Infinite scroll & pagination state
  let paginatedMedia: MediaItem[] = [];
  let totalMediaCount: number = 0;
  let pageSize = 180;
  let currentOffset = 0;
  let isLoadingMore = false;
  let hasMoreMedia = true;
  
  // Infinite scroll observer ref
  let sentinelRef: HTMLElement | null = null;
  let infiniteObserver: IntersectionObserver | null = null;'''

content, ok = patch("PATCH2", state_marker, content.split(state_marker)[0] + new_state + "\n" + state_marker if False else state_marker + "\n" + new_state, content)
if not ok:
    print("PATCH2 failed - state marker not found")

# ──────────────────────────────────────────────
# PATCH 3: Add new functions BEFORE onMount
# ──────────────────────────────────────────────
onmount_marker = "onMount(async () => {"

new_functions = """  // ── Login/Logout handlers ──
  async function handleLogin() {
    if (!loginUsername.trim()) { authError = "Username is required"; return; }
    try {
      await SaveProfile(loginUsername.trim(), loginFullname.trim() || loginUsername.trim());
      showLoginScreen = false;
      showDashboard = true;
      await loadPlatformDashboard();
    } catch (e) {
      authError = "Login failed: " + String(e);
    }
  }

  async function handleLogout() {
    try { await LogoutProfile(); } catch(e) {}
    showLoginScreen = false;
    showDashboard = false;
    paginatedMedia = [];
    selectedPlatform = null;
    currentOffset = 0;
    hasMoreMedia = true;
    mediaSources = {};
    mediaLoading = {};
  }

  // ── Dashboard loading ──
  async function loadPlatformDashboard() {
    try {
      const platforms = ['snapchat', 'instagram', 'facebook'];
      for (const p of platforms) {
        let stats: any;
        try { stats = await GetPlatformStats(p); } catch(e) { continue; }
        if (stats) {
          platformStatsList.push({
            id: p, name: p.charAt(0).toUpperCase() + p.slice(1), status: stats.mediaCount > 0 ? 'indexed' : 'empty',
            mediaCount: Number(stats.mediaCount || 0), imageCount: Number(stats.imageCount || 0), videoCount: Number(stats.videoCount || 0),
            zipCount: Number(stats.zipCount || 0), conversationCount: Number(stats.conversationCount || 0), 
            jsonFileCount: Number(stats.jsonFileCount || 0), yearsFound: Number(stats.yearsFound || 0)
          });
        }
      }
    } catch (e) {
      console.error("Failed to load dashboard stats:", e);
    }
  }

  async function selectPlatform(platform: string) {
    selectedPlatform = platform;
    selectedYearPaged = "all";
    paginatedMedia = [];
    totalMediaCount = 0;
    currentOffset = 0;
    hasMoreMedia = true;
    mediaSources = {};
    mediaLoading = {};
    await loadMediaBatch();
  }

  // ── Infinite scroll / pagination ──
  async function loadMediaBatch() {
    if (isLoadingMore || !hasMoreMedia || !selectedPlatform) return;
    isLoadingMore = true;
    
    try {
      const platform = selectedPlatform!;
      
      // Get total count first time
      if (currentOffset === 0) {
        try { 
          totalMediaCount = await GetMediaCount(platform, ""); 
        } catch(e) {}
      }
      
      const newItems: MediaItem[] = [];
      let batchOffset = currentOffset;
      let keepLoading = true;
      
      while (keepLoading) {
        const items = await GetMediaPaginated(
          platform, "",
          batchOffset,
          pageSize
        );
        
        if (!items || items.length === 0) break;
        
        newItems.push(...items);
        batchOffset += items.length;
        
        if (items.length < pageSize) break;
      }
      
      paginatedMedia.push(...newItems);
      currentOffset = batchOffset;
      hasMoreMedia = totalMediaCount > 0 ? currentOffset < totalMediaCount : currentOffset === 0;
      
      // Reset observer target
      setupInfiniteScroll();
    } catch (e) {
      console.error("Failed to load media:", e);
    } finally {
      isLoadingMore = false;
    }
  }

  function triggerLoadMore() {
    if (!isLoadingMore && hasMoreMedia) {
      currentOffset = paginatedMedia.length;
      loadMediaBatch();
    }
  }

  // ── Infinite scroll observer ──
  function setupInfiniteScroll() {
    if (infiniteObserver) {
      infiniteObserver.disconnect();
    }
    
    if (!sentinelRef) return;
    
    infiniteObserver = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting && hasMoreMedia && !isLoadingMore) {
        triggerLoadMore();
      }
    }, { rootMargin: "400px" });
    
    infiniteObserver.observe(sentinelRef);
  }

  // ── Settings persistence ──
  async function loadSettings() {
    try {
      const settings = await GetAppSettings();
      if (settings) pageSize = Number(settings.pagesize) || 180;
    } catch(e) {}
  }

  async function savePageSize(newSize: number) {
    pageSize = Math.max(30, Math.min(500, newSize));
    try { await SaveAppSettings({ pagesize: pageSize, loggedin: false }); } catch(e) {}
  }

"""

content, ok = patch("PATCH3", onmount_marker, new_functions + "onMount(async () => {", content)
if not ok:
    print("PATCH3 failed - onMount marker not found")

# ──────────────────────────────────────────────
# PATCH 4: Modify onMount to load settings
# ──────────────────────────────────────────────
onmount_body = '''    try {
      appState = await GetFrontendState();
      profileUsername = appState.profile.username ?? "";
      profileFullName = appState.profile.fullName ?? "";'''

new_onmount_body = '''    try {
      appState = await GetFrontendState();
      profileUsername = appState.profile.username ?? "";
      profileFullName = appState.profile.fullName ?? "";
      
      // Load user settings
      const settings = await GetAppSettings();
      if (settings) pageSize = Number(settings.pagesize) || 180;
      
      // Check login status and show appropriate screen
      if (!appState.profile.loggedIn) {
        showLoginScreen = true;
      } else {
        showDashboard = true;
        await loadPlatformDashboard();
      }'''

content, ok = patch("PATCH4", onmount_body, new_onmount_body, content)
if not ok:
    print("PATCH4 failed")

# ──────────────────────────────────────────────
# PATCH 5: Modify profile save/logout to update login state
# ──────────────────────────────────────────────
save_profile_patch = '''      appState = { ...appState, profile };
      profileOpen = false;'''

new_save_profile = '''      appState = { ...appState, profile };
      if (profile.loggedIn) {
        showLoginScreen = false;
        showDashboard = true;
        await loadPlatformDashboard();
      }
      profileOpen = false;'''

# We need to patch both saveProfileForm and switchUser which have same pattern
content = content.replace(save_profile_patch, new_save_profile, 2)
print(f"PATCH5: replace saveProfile/switchUser profile updates")

logout_patch = '''    try {
      const profile = await LogoutProfile();
      appState = { ...appState, profile };
      profileUsername = "";
      profileFullName = "";
      profileOpen = false;'''

new_logout = '''    try {
      const profile = await LogoutProfile();
      appState = { ...appState, profile };
      profileUsername = "";
      profileFullName = "";
      showLoginScreen = !profile.loggedIn;
      showDashboard = false;
      profileOpen = false;'''

content, ok = patch("PATCH5b", logout_patch, new_logout, content)

# ──────────────────────────────────────────────
# PATCH 6: Replace the template structure
# ──────────────────────────────────────────────
# The current flow is: error → loading → !summary → else (full app)
# New flow should be: error → loading → !loggedIn(login screen) → !summary(dashboard) → full app

error_section_start = '''{#if error}
  <main class="center">
    <section class="empty">
      <h1>Could not load the archive</h1>
      <p>{error}</p>
    </section>
  </main>
{:else if loading}'''

new_error_section = '''{#if error}
  <main class="center">
    <section class="empty">
      <h1>Could not load the archive</h1>
      <p>{error}</p>
    </section>
  </main>
{:else if loading}'''

content, ok = patch("PATCH6a", error_section_start, new_error_section, content)

# Now replace the !summary and else sections with login + dashboard logic
old_no_summary_else = '''{:else if !summary}
  <main class="center">
    <section class="empty">
      <h1>Indexing Snapchat export...</h1>
      <p>Your selected zips and indexed cache live in <code>{appState.storePath || "~/.mochila/database.sqlite"}</code>.</p>
      <p>
        {#if selected.length === 0}
          <button class="load-more" on:click={pickZips} disabled={selecting}>
            {selecting ? "Opening picker..." : "Choose export zip files"}
          </button>
        {:else}
          <button class="load-more" on:click={indexSelected} disabled={indexing}>
            {indexing ? "Indexing selected archives..." : `Index ${selected.length} zip${selected.length === 1 ? "" : "s"}`}
          </button>
        {/if}
      </p>
    </section>
  </main>
{:else}
  <header>'''

new_flow = '''{:else if showLoginScreen}
  <!-- LOGIN SCREEN -->
  <main class="center">
    <section class="login-card">
      <div style="text-align:center;margin-bottom:2rem;">
        <h1 style="font-size:2rem;margin-bottom:0.5rem;">Mochila Archive Viewer</h1>
        <p class="subtitle" style="max-width:300px;margin:0 auto;">Snapchat Data Explorer</p>
      </div>
      
      {#if authError}
        <p style="color:#ef4444;margin-bottom:1rem;text-align:center;">{authError}</p>
      {/if}
      
      <form on:submit|preventDefault={handleLogin}>
        <div class="form-group" style="margin-bottom:1rem;">
          <label for="login-username" style="display:block;margin-bottom:0.25rem;font-weight:600;">Username</label>
          <input 
            id="login-username"
            type="text" 
            bind:value={loginUsername}
            placeholder="Enter your username"
            autocomplete="username"
            style="width:100%;padding:0.5rem;border:1px solid #374151;border-radius:0.375rem;background:#1f2937;color:white;"
          />
        </div>
        
        <div class="form-group" style="margin-bottom:1.5rem;">
          <label for="login-fullname" style="display:block;margin-bottom:0.25rem;font-weight:600;">Full Name (optional)</label>
          <input 
            id="login-fullname"
            type="text" 
            bind:value={loginFullname}
            placeholder="Enter your full name"
            autocomplete="name"
            style="width:100%;padding:0.5rem;border:1px solid #374151;border-radius:0.375rem;background:#1f2937;color:white;"
          />
        </div>
        
        <button type="submit" class="btn-primary" style="width:100%;padding:0.75rem;font-size:1rem;">
          Sign In
        </button>
      </form>
    </section>
  </main>
{:else if showDashboard}
  <!-- PLATFORM DASHBOARD -->
  <main class="dashboard-main">
    <div style="max-width:960px;margin:0 auto;padding:2rem;">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:2rem;">
        <div>
          <h1 style="font-size:1.875rem;margin-bottom:0.25rem;">Mochila Archive Viewer</h1>
          <p class="subtitle">Select a platform to explore your archive</p>
        </div>
        <button on:click={handleLogout} class="btn-secondary" style="padding:0.5rem 1rem;">
          Logout
        </button>
      </div>
      
      <div class="platform-grid" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:1.5rem;">
        {#each platformStatsList as stat}
          <div 
            class="platform-card" 
            on:click={() => selectPlatform(stat.id)}
            style="background:#1f2937;border:1px solid #374151;border-radius:0.75rem;padding:1.5rem;cursor:pointer;transition:all 0.2s;"
          >
            <div style="display:flex;justify-content:space-between;align-items:start;margin-bottom:1rem;">
              <h2 style="font-size:1.25rem;text-transform:capitalize;">{stat.name}</h2>
              <span class="badge" style="padding:0.25rem 0.75rem;border-radius:9999px;font-size:0.75rem;{ stat.status === 'indexed' ? 'background:#065f46;color:#6ee7b7;' : 'background:#374151;color:#9ca3af;' }">
                {stat.status}
              </span>
            </div>
            
            <div style="display:grid;grid-template-columns:repeat(2,1fr);gap:0.75rem;margin-top:1rem;">
              <div>
                <div style="font-size:1.5rem;font-weight:700;color:#f3f4f6;">{stat.mediaCount.toLocaleString()}</div>
                <div style="font-size:0.75rem;color:#9ca3af;">Media Items</div>
              </div>
              <div>
                <div style="font-size:1.25rem;font-weight:600;color:#f3f4f6;">{stat.zipCount}</div>
                <div style="font-size:0.75rem;color:#9ca3af;">Zips</div>
              </div>
              <div>
                <div style="font-size:1rem;font-weight:600;color:#60a5fa;">{stat.imageCount.toLocaleString()}</div>
                <div style="font-size:0.75rem;color:#9ca3af;">Photos</div>
              </div>
              <div>
                <div style="font-size:1rem;font-weight:600;color:#f87171;">{stat.videoCount.toLocaleString()}</div>
                <div style="font-size:0.75rem;color:#9ca3af;">Videos</div>
              </div>
            </div>
            
            {#if stat.conversationCount > 0}
              <div style="margin-top:1rem;padding-top:0.75rem;border-top:1px solid #374151;font-size:0.875rem;color:#9ca3af;">
                {stat.conversationCount.toLocaleString()} conversations · {stat.jsonFileCount.toLocaleString()} JSON files
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  </main>
{:else if !summary}
  <main class="center">
    <section class="empty">
      <h1>Indexing Snapchat export...</h1>
      <p>Your selected zips and indexed cache live in <code>{appState.storePath || "~/.mochila/database.sqlite"}</code>.</p>
      <p>
        {#if selected.length === 0}
          <button class="load-more" on:click={pickZips} disabled={selecting}>
            {selecting ? "Opening picker..." : "Choose export zip files"}
          </button>
        {:else}
          <button class="load-more" on:click={indexSelected} disabled={indexing}>
            {indexing ? "Indexing selected archives..." : `Index ${selected.length} zip${selected.length === 1 ? "" : "s"}`}
          </button>
        {/if}
      </p>
    </section>
  </main>
{:else}
  <header>'''

content, ok = patch("PATCH6", old_no_summary_else, new_flow, content)
if not ok:
    print("PATCH6 failed - dashboard flow replacement")

# ──────────────────────────────────────────────
# PATCH 7: Update indexSelected to show dashboard after indexing
# ──────────────────────────────────────────────
index_selected_patch = '''      mediaSources = {};
      await ensureMediaSources(media.slice(0, 60));
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);'''

new_index_selected = '''      mediaSources = {};
      await ensureMediaSources(media.slice(0, 60));
      
      // After indexing, refresh dashboard if visible
      if (showDashboard && selectedPlatform === activePlatform) {
        await loadPlatformDashboard();
      }
    } catch (caught) {
      error = caught instanceof Error ? caught.message : String(caught);'''

content, ok = patch("PATCH7", index_selected_patch, new_index_selected, content)

# ──────────────────────────────────────────────
# PATCH 8: Update settings modal with page size control  
# ──────────────────────────────────────────────
settings_patch = '''{#if settingsOpen}'''

new_settings = '''// Page size for gallery display
$: pageSizeLabel = pageSize + " items per page";

{#if settingsOpen}'''

content, ok = patch("PATCH8", settings_patch, new_settings, content)

# Find and add page size input to settings modal
settings_content_patch = '''        <div class="setting-row">
          <span>Theme</span>
          <select bind:value={theme}>
            <option value="dark">Dark</option>'''

new_settings_row = '''        <div class="setting-row">
          <label for="page-size-input" style="display:flex;flex-direction:column;gap:0.25rem;">
            <span>Items per page (gallery)</span>
            <input 
              id="page-size-input"
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
        
        <div class="setting-row">
          <span>Theme</span>
          <select bind:value={theme}>
            <option value="dark">Dark</option>'''

content, ok = patch("PATCH8b", settings_content_patch, new_settings_row, content)

# ──────────────────────────────────────────────
# PATCH 9: Add infinite scroll sentinel + "load more" button after visibleMedia
# ──────────────────────────────────────────────
load_more_btn_patch = '''        {#if visibleMedia.length < filteredMedia.length}
          <div style="text-align:center;padding:2rem;">
            <button class="load-more" on:click={loadMore}>
              Load more
            </button>
          </div>'''

new_load_more_btn = '''        {#if paginatedMedia.length > 0 && paginatedMedia.length < totalMediaCount}
          <div style="text-align:center;padding:1rem;">
            <button class="load-more" on:click={triggerLoadMore} disabled={isLoadingMore}>
              {isLoadingMore ? "Loading..." : "Load more"} ({paginatedMedia.length} of {totalMediaCount.toLocaleString()})
            </button>
          </div>
        {/if}
        
        <!-- Infinite scroll sentinel -->
        <div bind:this={sentinelRef} style="height:1px;"></div>'''

content, ok = patch("PATCH9", load_more_btn_patch, new_load_more_btn, content)

# ──────────────────────────────────────────────
# PATCH 10: Update visibleMedia to use paginated data for gallery view
# ──────────────────────────────────────────────
visible_media_reactive = '''  $: visibleMedia = searchableMedia.slice(0, visibleLimit);'''

new_visible_media = '''  // Use paginated media when in infinite scroll mode, otherwise original slicing
  $: visibleMedia = selectedPlatform ? paginatedMedia : searchableMedia.slice(0, visibleLimit);
  
  // Override filteredMedia for gallery view to use paginated data
  $: galleryMedia = selectedPlatform ? paginatedMedia : filteredMedia;'''

content, ok = patch("PATCH10", visible_media_reactive, new_visible_media, content)

# ──────────────────────────────────────────────
# PATCH 11: Add cleanup for infinite observer
# ──────────────────────────────────────────────
write_file(content)
print("\n=== Patching complete! ===")
print("Run 'npm run build' to verify compilation.")
