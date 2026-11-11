#!/usr/bin/env python3
"""Consistent SQLite backup, including committed WAL contents."""
import pathlib
import sqlite3
import sys
if len(sys.argv) != 3:
    raise SystemExit('Usage: python3 scripts/backup.py SOURCE.db DESTINATION.db')
source, destination = map(pathlib.Path, sys.argv[1:])
if not source.is_file() or source.resolve() == destination.resolve():
    raise SystemExit('Source must exist and destination must differ')
if destination.exists():
    raise SystemExit('Choose a new destination; refusing to overwrite a backup')
destination.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
with sqlite3.connect(source.resolve().as_uri() + '?mode=ro', uri=True) as src:
    with sqlite3.connect(destination) as dst:
        src.backup(dst)
destination.chmod(0o600)
print(destination)
