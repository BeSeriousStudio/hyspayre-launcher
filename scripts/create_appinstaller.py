#!/usr/bin/env python3
"""Create a Windows App Installer update descriptor for a *signed* uploaded MSIX.

Run only AFTER signing the exact MSIX and publishing it over HTTPS.
This does not sign the package or make it trusted.
"""
import argparse
import sys
import xml.etree.ElementTree as ET
from pathlib import Path
from xml.sax.saxutils import escape

VERSION = '1.2.5.0'
FILENAME = 'HyspayreLauncher_1.2.5.0_x64.msix'

def main():
    ap = argparse.ArgumentParser(description='Generate Hyspayre.appinstaller after trusted MSIX signing')
    ap.add_argument('--identity', required=True)
    ap.add_argument('--publisher', required=True)
    ap.add_argument('--base-url', default='https://hyspayre.com/downloads/')
    ap.add_argument('--output', default='dist/Hyspayre.appinstaller')
    args = ap.parse_args()
    if not args.publisher.startswith('CN='):
        ap.error('Publisher must match MSIX certificate identity (CN=...)')
    if not args.base_url.startswith('https://') or not args.base_url.endswith('/'):
        ap.error('Use HTTPS base URL ending with /')
    if not args.identity.strip():
        ap.error('Missing MSIX identity')
    if args.identity == 'Hyspayre.Development' or args.publisher == 'CN=Hyspayre-Development':
        ap.error('Do not distribute a DEV identity. Configure the real publisher and sign with a trusted certificate.')
    esc = lambda s: escape(s, {'"': '&quot;'})
    uri = esc(args.base_url+'Hyspayre.appinstaller')
    msixuri = esc(args.base_url+FILENAME)
    manifest = f'''<?xml version="1.0" encoding="utf-8"?>
<AppInstaller xmlns="http://schemas.microsoft.com/appx/appinstaller/2017/2"
              Uri="{uri}" Version="{VERSION}">
  <MainPackage Name="{esc(args.identity)}" Publisher="{esc(args.publisher)}"
               Version="{VERSION}" ProcessorArchitecture="x64"
               Uri="{msixuri}" />
  <UpdateSettings><OnLaunch HoursBetweenUpdateChecks="12" /></UpdateSettings>
</AppInstaller>
'''
    ET.fromstring(manifest)
    dest = Path(args.output)
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(manifest, encoding='utf-8')
    print('Created:', dest)
    print('IMPORTANT: upload only after the referenced MSIX is correctly signed and public via HTTPS')

if __name__=='__main__': main()
