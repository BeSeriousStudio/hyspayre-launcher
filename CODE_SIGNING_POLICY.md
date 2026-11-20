# Code signing policy — Hyspayre Launcher

**Status:** Planned. The project has **not** yet been approved by SignPath Foundation, and the published 0.2.3 build is not claimed to have an Authenticode signature.

If approved, the project intends to use: **Free code signing provided by SignPath.io, certificate by SignPath Foundation**.

## Scope

This policy applies only to the Windows Hyspayre Launcher and its Windows installer built from the public source repository:
https://github.com/BeSeriousStudio/hyspayre-launcher

The proprietary Hyspayre Godot game, external game downloads, and the private update-catalog signing tool are **not** within this open-source signing scope and must not be bundled into signed artifacts.

## Maintainer roles

- **Authors / committers:** [BeSeriousStudio](https://github.com/BeSeriousStudio) (repository maintainer; verify this is the correct GitHub account).
- **Reviewers:** [BeSeriousStudio](https://github.com/BeSeriousStudio). Pull requests from outside contributors must be reviewed by a trusted maintainer before merging.
- **Approvers:** [BeSeriousStudio](https://github.com/BeSeriousStudio). A trusted maintainer must approve **every** signing request.

Maintainers must enable multi-factor authentication for GitHub and SignPath. Source changes, especially to build scripts, installer and CI workflows, must be reviewed before releases.

## Build and release

Source: public GitHub repository and reviewed revision. Existing CI: `.github/workflows/build.yml`. It currently creates **unsigned** Windows artifacts. No public signing is yet configured.

Once SignPath approval and configuration are granted, official signed artifacts will be produced from traceable GitHub-hosted CI artifacts, with manually approved signing requests and the product/version metadata required by SignPath. Releases will be posted on GitHub Releases and/or the project's documented download page.

The launcher's Ed25519 **update catalog signature** is a separate security mechanism and is **not** a substitute for Authenticode. The private catalog signing key is never committed to GitHub or included in CI. The final production update catalog must use the hash of the already Authenticode-signed launcher binary.

## Privacy

See [Privacy information for the desktop launcher](PRIVACY.md) and the [Hyspayre legal/privacy page](https://hyspayre.com/rechtliches.html).

The launcher communicates with Hyspayre servers to retrieve update information and optional game downloads, and with Supabase for the sign-in and community features that users choose to access. Details and applicable third-party policies must be disclosed to users before the relevant data flows.

Third-party service privacy policies: [Supabase](https://supabase.com/privacy) and, where applicable, the project's hosting provider. The maintainer must verify that the public privacy notice accurately reflects production behavior.

## Security issues

See [SECURITY.md](SECURITY.md). Report security problems privately; never publicly post credentials or user records.

**Before submitting to SignPath Foundation:** verify the artwork licensing, installer privacy notice/choices, project release, CI provenance, and this policy's maintainer assignments against actual practice. The Foundation alone decides eligibility.
