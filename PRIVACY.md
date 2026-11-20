# Hyspayre Launcher — privacy information

> **DRAFT — MUST BE REVIEWED AND COMPLETED BY THE PROJECT OPERATOR BEFORE PUBLIC RELEASE.** This document describes behavior identified in the Hyspayre Launcher 0.2.3 source. It is not a substitute for a complete, legally verified privacy notice.

## Controller and contact

Project: Hyspayre Launcher / BeSeriousStudio.

**ACTION REQUIRED:** Identify the legally responsible controller, business/contact address and a contact email in the published privacy notice. Ensure the page at https://hyspayre.com/rechtliches.html is current and reachable.

## Network connections and data

The launcher uses the following services:

1. **Hyspayre website / update infrastructure** (`https://hyspayre.com`): online game/update catalog, release downloads and some artwork. When network resources are requested, the server/host may process technical connection data such as IP address, timestamp and HTTP metadata. Verify actual server logging and retention with your hosting provider.
2. **Supabase** (`https://amgpdukqacxogafqpmho.supabase.co`): optional account sign-in, account/profile retrieval, player search and friend management. Account credentials are transmitted to Supabase over HTTPS when the user signs in. The launcher displays profile identifiers, public names, avatars, ranks and friendship relationships as returned by authorized APIs. Account and friendship information is shared with the Hyspayre website and Godot game.
3. **Local storage on Windows**: when 'stay signed in' is enabled, session tokens are protected with Windows DPAPI and stored at `%LOCALAPPDATA%\HyspayreLauncher\session.dpapi`. The application also stores local installation settings and game-related state. The installer removes application binaries but leaves installed games and some local settings behind so they are not unexpectedly deleted. The user can remove the remaining local data separately.

The supplied source code also verifies downloaded update signatures and SHA-256 checksums; these checks do not themselves require sending a private signing key to an external service.

## User choices

- An account is not required solely to view or install the launcher; logging in is a user-initiated feature.
- 'Stay signed in' is optional. Logging out should remove the locally stored account session; users should verify this in the shipped version.
- Game installations and launcher updates are initiated from the launcher UI.

**ACTION REQUIRED:** Confirm whether the launcher fetches update metadata automatically at startup. If network data is sent without a specific user request, disclose the transfer during setup and provide an installation option to disable it where SignPath Foundation's privacy conditions apply. Merely publishing this document does not implement the required installer UI.

## Processing details still to be completed

**ACTION REQUIRED:** Before treating this as the final user-facing notice, document the actual GDPR lawful bases, categories of personal data, retention periods, deletion procedure, recipients and cross-border transfers, data-subject rights, contact route, and whether crash/telemetry/analytics are collected. Verify that the website, game and launcher notices do not contradict each other.

Additional service policies: [Supabase privacy policy](https://supabase.com/privacy). Add the relevant hosting provider's policy if users are affected.

Last drafted: 2026-11-20.
