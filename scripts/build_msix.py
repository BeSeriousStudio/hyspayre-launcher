#!/usr/bin/env python3
"""Stage a Windows Desktop MSIX package. Run MakeAppx.exe on Windows afterwards.

This script deliberately does NOT sign anything. Store submissions are signed by
Microsoft after acceptance. For direct .msix/.appinstaller distribution, use a
trusted code-signing certificate on the MSIX before putting files on the site.
"""
from __future__ import annotations

import argparse
import re
import shutil
import xml.etree.ElementTree as ET
from pathlib import Path
from xml.sax.saxutils import escape

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
PACKAGE_VERSION = '1.2.5.0'  # Store forbids a zero first version component.
PACKAGE_FILE = 'HyspayreLauncher_1.2.5.0_x64.msix'


def icon(target: Path, size: tuple[int, int], icon_px: int | None = None) -> None:
    original = Image.open(ROOT / 'cmd/launcher/ui/assets/hyspayre-icon-512.png').convert('RGBA')
    w, h = size
    # transparent icon/Windows tiles, with the same dark-navy Hyspayre visual identity
    img = Image.new('RGBA', size, (8, 15, 27, 255))
    margin = max(3, min(w, h) // 10)
    display = icon_px or min(w-2*margin, h-2*margin)
    display = max(1, min(display, w, h))
    square = original.resize((display, display), Image.Resampling.LANCZOS)
    img.alpha_composite(square, ((w-display)//2, (h-display)//2))
    img.convert('RGB').save(target, 'PNG', optimize=True)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--identity', default='Hyspayre.Development', help='Exact Partner Center package/identity name')
    ap.add_argument('--publisher', default='CN=Hyspayre-Development', help='Exact Partner Center Publisher DN')
    ap.add_argument('--out', default=str(ROOT / 'msix/staging'))
    ap.add_argument('--exe', default=str(ROOT / 'dist/Hyspayre-Launcher.exe'))
    args = ap.parse_args()
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]{2,49}', args.identity):
        ap.error('Invalid package identity; use the exact Partner Center value')
    if not args.publisher.startswith('CN=') or len(args.publisher) > 400:
        ap.error('Publisher must be the exact certificate distinguished name from Partner Center, beginning with CN=')
    if not Path(args.exe).is_file():
        ap.error('Missing compiled launcher EXE; first run python scripts/build.py')
    out = Path(args.out).resolve()
    if out.exists():
        shutil.rmtree(out)
    (out / 'Assets').mkdir(parents=True)
    shutil.copy2(args.exe, out / 'Hyspayre-Launcher.exe')
    icon(out / 'Assets/Square44x44Logo.png', (44, 44))
    icon(out / 'Assets/Square150x150Logo.png', (150, 150))
    icon(out / 'Assets/StoreLogo.png', (50, 50))
    icon(out / 'Assets/Wide310x150Logo.png', (310, 150), 130)
    icon(out / 'Assets/Square310x310Logo.png', (310, 310))
    # XML escape attribute values. MSIX package identity/publisher MUST later
    # match Partner Center precisely or Store submission will be rejected.
    name, publisher = escape(args.identity, {'"':'&quot;'}), escape(args.publisher, {'"':'&quot;'})
    xml = f'''<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
         xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
         xmlns:uap10="http://schemas.microsoft.com/appx/manifest/uap/windows10/10"
         xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
         IgnorableNamespaces="uap uap10 rescap">
  <Identity Name="{name}" Publisher="{publisher}" Version="{PACKAGE_VERSION}" ProcessorArchitecture="x64" />
  <Properties>
    <DisplayName>Hyspayre Launcher</DisplayName>
    <PublisherDisplayName>Hyspayre</PublisherDisplayName>
    <Description>Hyspayre Spielebibliothek, Updates und Community</Description>
    <Logo>Assets\\StoreLogo.png</Logo>
  </Properties>
  <Resources><Resource Language="de-DE"/><Resource Language="en-US"/></Resources>
  <Dependencies><TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.19041.0" MaxVersionTested="10.0.26100.0" /></Dependencies>
  <Applications>
    <Application Id="HyspayreLauncher" Executable="Hyspayre-Launcher.exe"
                 uap10:RuntimeBehavior="packagedClassicApp" uap10:TrustLevel="mediumIL">
      <uap:VisualElements DisplayName="Hyspayre Launcher"
                          Description="Spiele und Community im Königreich"
                          BackgroundColor="#080f1b"
                          Square150x150Logo="Assets\\Square150x150Logo.png"
                          Square44x44Logo="Assets\\Square44x44Logo.png">
        <uap:DefaultTile Wide310x150Logo="Assets\\Wide310x150Logo.png"
                         Square310x310Logo="Assets\\Square310x310Logo.png" />
      </uap:VisualElements>
    </Application>
  </Applications>
  <Capabilities><rescap:Capability Name="runFullTrust" /></Capabilities>
</Package>
'''
    ET.fromstring(xml)
    (out / 'AppxManifest.xml').write_text(xml, encoding='utf-8')
    print(f'Staged: {out}')
    print(f'MSIX version {PACKAGE_VERSION}, identity={args.identity}, publisher={args.publisher}')
    print('UNSIGNED package; NOT installable via website until signed by trusted certificate.')
    if args.identity == 'Hyspayre.Development' or args.publisher == 'CN=Hyspayre-Development':
        print('DEV IDENTITY – DO NOT SUBMIT TO THE MICROSOFT STORE.')


if __name__ == '__main__':
    main()
