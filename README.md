# Hyspayre Launcher

Open-source-ready Windows launcher for the Hyspayre universe. Multi-game
installer, cryptographically signed update catalog, personal Hyspayre account,
player search and friend requests (using the same Supabase RPCs as the website).

**Current version: 0.2.3** · Go 1.23 · JavaScript, HTML and CSS · MIT code license.

## Build from source

Prerequisites: Go 1.23+, Python 3.10+. No npm packages or third-party Go libraries.

```bash
python scripts/build.py
```

This produces `dist/Hyspayre-Launcher.exe` and `dist/Hyspayre-Setup.exe`.
The build program generates the embedded installer package and pins its SHA-256.
A release is **not signed** automatically. Windows SmartScreen may warn.

## Social features

- Local login form directly to Supabase Auth using the public publishable API key.
- Profile and rank from `bss_me()`; list from `bss_social_friends()`.
- Live search from `bss_social_search_live()` (1 character, paginated), with a limited fallback to `bss_social_search()` if SQL 08 is not installed (2+ characters; up to 20 matches).
- Public profile from `bss_social_profile()` and friendship changes using
  `bss_social_action()`.
- Profiles and friendships are **shared with hyspayre.com and the Godot game**.
- For unlimited live results and one-character searches, run the bundled `Supabase/08_live_player_search.sql` script. The launcher can search up to 20 names via SQL 07 without it. No new tables or privileged keys.

**Authentication security:** credentials are sent from the local 127.0.0.1-only
interface to Supabase over HTTPS. The native client stores neither plaintext
passwords nor secrets in the UI. If "stay signed in" is enabled, Windows DPAPI
protects the session tokens for the logged-in Windows user in
`%LOCALAPPDATA%/HyspayreLauncher/session.dpapi`. The user can log out to revoke
and delete them. Other operating systems currently do not support remembered login.
The UI API requires a random session secret and verifies localhost Origin and Host.

## Repository boundaries

This repository intentionally **does not contain** the private Ed25519 catalog
signing key, the commercial Godot games, the private Publisher tool, Supabase
service-role keys, SMTP credentials or signing certificates. The embedded public
Ed25519 verification key and Supabase **publishable** key are intentionally public.

The launcher displays the original Hyspayre **blue diamond brand mark** from
`cmd/launcher/ui/assets/hyspayre-emblem.png`; the matching installer icon lives
at `setup/hyspayre.ico`. Brand artwork is **not automatically MIT-licensed**
merely because the surrounding code is. The rights holder must explicitly
approve an open-use license for the image before redistributing it on a public
open-source repository or applying to SignPath. Other brand artwork is fetched
from the public Hyspayre website at runtime; the Godot game remains private.

## Releasing

1. Build both Windows binaries using `scripts/build.py`.
2. In the *private* Hyspayre Publisher, keep the original private signing key
   offline. Sign a new `launcher/catalog.json` with version 0.2.3 and the SHA-256
   of `Hyspayre-Launcher.exe`.
3. Upload `Hyspayre-Setup.exe` and `Hyspayre-Launcher.exe` to
   `https://hyspayre.com/downloads/`, and **last** upload the signed catalog to
   `https://hyspayre.com/launcher/catalog.json`.
4. Existing clients learn of version 0.2.3 from the signed catalog. New users
   download the installer. Never upload the private Publisher key.

**If games have already been published:** do not overwrite the live catalog with
an old template. Download the current production catalog, edit it privately using
the publisher and re-sign it, preserving all game entries and versions.

## SignPath Foundation

This public source is designed to be auditable, but free SignPath Foundation
acceptance is **not guaranteed**. The foundation assesses licensing, build
provenance and project reputation. Publish a real GitHub history, configure
reviewed CI builds and apply separately. Do not add private keys to CI or commits.

## Limits

- The Launcher itself is a native Windows executable hosting a localhost UI in
  the installed browser's app mode. No custom WebView2 bundling is needed.
- Accounts are currently authenticated separately from the website; a browser
  login does not automatically log the desktop app in.
- Account **editing** continues on the website. Profile viewing, friend search,
  requests and friendships are in the Launcher.
- Catalog and installer signing are separate: Ed25519 protects updates but does
  **not** eliminate SmartScreen warnings. The Windows EXE/Setup needs Authenticode
  signing or Microsoft Store distribution to address those warnings.


## Installer UI

Version 0.2.3 adds a branded Windows welcome screen for the setup executable.
It uses the official Hyspayre logo and artwork, but still installs per-user
without administrator rights. If PowerShell WPF is unavailable, the installer
falls back to simple Windows dialog boxes.

## Code signing policy

[Code signing policy](CODE_SIGNING_POLICY.md) — Windows signing through SignPath Foundation is planned, **not approved or enabled** for this release.

## Privacy

[Desktop launcher privacy information](PRIVACY.md) — **draft requiring operator review before public release**, and [Hyspayre legal/privacy page](https://hyspayre.com/rechtliches.html).

## Artwork licensing

See [BRAND_ASSETS.md](BRAND_ASSETS.md). Rights and open-license choices for the bundled brand imagery are not finalized.
