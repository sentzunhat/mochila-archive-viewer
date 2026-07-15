#!/usr/bin/env python3
"""
Patch App.svelte to add:
1. Login/welcome screen
2. Platform dashboard with stats
3. Infinite scroll pagination
4. Configurable page size settings
"""

import os

# Read the current file - work around any tool artifacts by reading directly via subprocess
import subprocess
result = subprocess.run(
    ['cat', 'src/App.svelte'],
    capture_output=True,
    text=True,
    cwd='/Users/beltrd/Desktop/projects/sentzunhat/mochila-archive-viewer/src/frontend'
)

if result.returncode != 0:
    print(f"ERROR reading file: {result.stderr}")
    exit(1)

content = result.stdout
print(f"Read {len(content)} chars from App.svelte")

# Step 1: Add new imports
old_import_block = '''  import {
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
  } from "../wailsjs/go/appshell/App.js";'''

new_import_block = '''  import {
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
  } from "../wailsjs/go/appshell/App.js";'''

if old_import_block in content:
    content = content.replace(old_import_block, new_import_block)
    print("✓ Step 1: Added GetMediaPaginated, GetMediaCount, GetPlatformStats, GetAppSettings, SaveAppSettings imports")
else:
    print("✗ Step 1: Could not find import block to patch")

# Step 2: Add new types after existing type definitions
type_addition = '''  // Dashboard and pagination types
  type PlatformStatItem = { id: string; name: string; status: string; mediaCount: number; imageCount: number; videoCount: number; zipCount: number; conversationCount: number; jsonFileCount: number; yearsFound: number };
  type AppSettingsData = { pagesize: number; loggedin: boolean };'''

# Find the line after the existing type definitions
types_section = '''  type IndexSummary = {
    platform: string;
    mediaCount: number;
    zipCount: number;
    years: Record<string, number>;
    types: Record<string, number>;
    categories: Record<string, number>;'''

if types_section in content:
    content = content.replace(types_section, types_section + "\n" + type_addition)
    print("✓ Step 2: Added Dashboard/Pagination types")
else:
    print("✗ Step 2: Could not find IndexSummary type definition")

# Write the modified content back
output_path = os.path.join('/Users/beltrd/Desktop/projects/sentzunhat/mochila-archive-viewer/src/frontend', 'src/App.svelte')
with open(output_path, 'w') as f:
    f.write(content)

print(f"\nWrote patched file ({len(content)} chars)")
print("Done with basic imports and types!")
