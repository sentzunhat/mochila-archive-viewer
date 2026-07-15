#!/usr/bin/env python3
"""Fix PATCH8b (page size settings) and PATCH9 (infinite scroll sentinel)."""
import subprocess

SVELTE = "/Users/beltrd/Desktop/projects/sentzunhat/mochila-archive-viewer/src/frontend/src/App.svelte"

with open(SVELTE, 'r') as f:
    content = f.read()

lines = content.split('\n')

# Find actual patterns
for i, line in enumerate(lines):
    if 'Theme' in line and not line.strip().startswith('//'):
        print(f"Line {i+1}: {line.rstrip()}")
    if 'load-more' in line and 'Load more' in line:
        print(f"Line {i+1}: {line.rstrip()}")
