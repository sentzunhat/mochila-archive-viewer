#!/usr/bin/env python3
"""
Patch src/frontend/src/App.svelte to add:
 1) Login/welcome screen (entry point before anything else)
 2) Platform selection dashboard with stats
 3) Infinite scroll pagination for gallery
 4) Configurable page size settings

Reads the CURRENT file state from disk, makes surgical patches, writes back.
"""
import subprocess, re, os

WORKSPACE = "/Users/beltrd/Desktop/projects/sentzunhat/mochila-archive-viewer"
FRONTEND = os.path.join(WORKSPACE, "src/frontend")
SVELTE_FILE = os.path.join(FRONTEND, "src/App.svelte")

# ─── 1. Read the file ───
result = subprocess.run(["cat", SVELTE_FILE], capture_output=True, text=True)
if result.returncode != 0:
    raise RuntimeError(f"Cannot read {SVELTE_FILE}: {result.stderr}")

content = result.stdout
print(f"[1/6] Read {len(content)} chars ({content.count(chr(10))} lines)")

# ─── 2. Add new Wails imports ───
old_imports = """import {
    GetFrontendState,
    ActiveUserProfile,
    AvailableUsers,
    SelectUser,
    GetPlatformSnapshot,
    GetMedia,
    GetConversations,
    GetConversation,
    GetJSONPreview,
    GetMediaSource,
    SelectArchiveZips,
    IndexArchives,
    SaveProfile,
    LogoutProfile,
  } from "../wailsjs/go/appshell/App.js";"""

new_imports = """import {
    GetFrontendState,
    ActiveUserProfile,
    AvailableUsers,
    SelectUser,
    GetPlatformSnapshot,
    GetMedia,
    GetMediaPaginated,
    GetMediaCount,
    GetPlatformStats,
    GetAppSettings,
    SaveAppSettings,
    GetConversations,
    GetConversation,
    GetJSONPreview,
    GetMediaSource,
    SelectArchiveZips,
    IndexArchives,
    SaveProfile,
    LogoutProfile,
  } from "../wailsjs/go/appshell/App.js";"""

if old_imports in content:
    content = content.replace(old_imports, new_imports)
    print("[2/6] ✓ Added pagination/stats/settings imports")
else:
    # Try with different whitespace - check what's actually there
    lines = content.split('\n')[:40]
    for i, line in enumerate(lines):
        if 'GetMedia,' in line or 'GetFrontendState' in line:
            print(f"  Line {i+1}: {repr(line)}")
    raise RuntimeError("Import block not found - file may already be patched")

# ─── 3. Add new type definitions after existing types ───
# Find the last type definition and append ours after it
type_patch = """
  // Dashboard & pagination types
  type PlatformStatItem = { 
    id: string; name: string; status: string; 
    mediaCount: number; imageCount: number; videoCount: number; 
    zipCount: number; conversationCount: number; jsonFileCount: number; yearsFound: number 
  };"""

# Look for where type definitions end (before const declarations)
type_marker = "const visibleMedia: MediaItem[] = [];"
if type_marker in content:
    content = content.replace(type_marker, type_patch + "\n\n" + type_marker)
    print("[3/6] ✓ Added dashboard/pagination types")
else:
    # Check if we need different approach
    raise RuntimeError("Could not find type insertion point - file may be corrupted or already patched")

# ─── 4. Add new state variables ───
# Insert after the existing state variable declarations (e.g., after visibleLimit)
state_insert_marker = "let visibleLimit = 180;"
new_state_vars = """
  // Login & dashboard state
  let showLoginScreen: boolean = false;
  let loginUsername = "";
  let loginFullname = "";
  let authError = "";
  
  // Platform dashboard state
  let showDashboard = false;
  let platformStatsList: PlatformStatItem[] = [];
  let selectedPlatform: string | null = null;
  let selectedYear: string = "all";
  
  // Infinite scroll & pagination state
  let mediaItems: MediaItem[] = [];
  let totalMediaCount: number = 0;
  let pageSize = 180;
  let currentOffset = 0;
  let isLoadingMore = false;
  let hasMoreMedia = true;
  
  // Infinite scroll observer
  let sentinelRef: HTMLElement | null = null;
  let infiniteObserver: IntersectionObserver | null = null;"""

if state_insert_marker in content:
    content = content.replace(state_insert_marker, state_insert_marker + new_state_vars)
    print("[4/6] ✓ Added login/dashboard/pagination state variables")
else:
    raise RuntimeError("Could not find state insertion point")

# ─── 5. Add new functions before onMount ───
# Find the onMount function and add our handlers BEFORE it
onmount_marker = "onMount(async () => {"
new_functions = """
  // ── Login/Logout handlers ──
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
    mediaItems = [];
    selectedPlatform = null;
    currentOffset = 0;
    hasMoreMedia = true;
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
    selectedYear = "all";
    mediaItems = [];
    currentOffset = 0;
    hasMoreMedia = true;
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
          totalMediaCount = await GetMediaCount(platform, selectedYear === "all" ? "" : selectedYear); 
        } catch(e) {}
      }
      
      const newItems: MediaItem[] = [];
      let batchOffset = currentOffset;
      let keepLoading = true;
      
      while (keepLoading) {
        const items = await GetMediaPaginated(
          platform,
          selectedYear === "all" ? "" : selectedYear,
          batchOffset,
          pageSize
        );
        
        if (!items || items.length === 0) break;
        
        newItems.push(...items);
        batchOffset += items.length;
        
        if (items.length < pageSize) break;
      }
      
      mediaItems.push(...newItems);
      currentOffset = batchOffset;
      hasMoreMedia = currentOffset < totalMediaCount || currentOffset === 0;
      
      // Reset observer target
      setupInfiniteScroll();
    } catch (e) {
      console.error("Failed to load media:", e);
    } finally {
      isLoadingMore = false;
    }
  }

  function loadMoreMedia() {
    currentOffset = mediaItems.length;
    hasMoreMedia = true;
    loadMediaBatch();
  }

  // ── Infinite scroll observer ──
  function setupInfiniteScroll() {
    if (infiniteObserver) {
      infiniteObserver.disconnect();
    }
    
    if (!sentinelRef) return;
    
    infiniteObserver = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting && hasMoreMedia && !isLoadingMore) {
        loadMediaBatch();
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

  async function savePageSetting(newSize: number) {
    pageSize = Math.max(30, Math.min(500, newSize));
    try { await SaveAppSettings({ pagesize: pageSize, loggedin: false }); } catch(e) {}
  }

  // ── Override visibleMedia reactive to use paginated data ──
  
`;

if onmount_marker in content:
    content = content.replace(onmount_marker, new_functions + "onMount(async () => {")
    print("[5/6] ✓ Added login/dashboard/pagination/settings functions")
else:
    raise RuntimeError("Could not find onMount insertion point")

# ─── 6. Write the modified file ───
with open(SVELTE_FILE, 'w') as f:
    f.write(content)

print(f"[6/6] ✓ Wrote patched file ({content.count(chr(10))} lines)")
print("\n=== Patch complete! Run `npm run build` to verify. ===")
