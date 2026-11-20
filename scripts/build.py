#!/usr/bin/env python3
"""Build both Windows binaries using only Go 1.23 and Python 3 stdlib.

No secret signing keys, IONOS credentials or game project source needed.
This creates UNSIGNED binaries. SignPath integration is separate after approval.
"""
import os
import hashlib
import subprocess
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / 'dist'
SETUP = ROOT / 'setup'
DIST.mkdir(parents=True, exist_ok=True)

def go_build(directory, destination):
    env = os.environ.copy()
    env.update(GOOS='windows', GOARCH='amd64', CGO_ENABLED='0')
    subprocess.run(['go','build','-trimpath','-ldflags=-s -w -H=windowsgui','-o',str(destination),'./'],cwd=directory,env=env,check=True)

subprocess.run(['go','test','./...'],cwd=ROOT,check=True)
launcher = DIST / 'Hyspayre-Launcher.exe'
go_build(ROOT / 'cmd/launcher', launcher)
sha = hashlib.sha256(launcher.read_bytes()).hexdigest()
with zipfile.ZipFile(SETUP/'payload.zip','w',zipfile.ZIP_DEFLATED,compresslevel=9) as z:
    z.write(launcher,'Hyspayre-Launcher.exe')
    z.write(SETUP/'hyspayre.ico','hyspayre.ico')
(SETUP/'launcher_hash_generated.go').write_text('package main\nconst expectedLauncherSHA256 = "'+sha+'"\n',encoding='utf-8')
go_build(SETUP, DIST / 'Hyspayre-Setup.exe')
print('Windows binaries:',DIST)
print('Launcher SHA-256:',sha)
print('NOTE: unsigned binaries can still trigger Windows SmartScreen.')
