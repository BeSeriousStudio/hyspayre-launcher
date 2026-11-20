# Hyspayre Launcher 0.2.3 — SignPath readiness checklist

- [ ] Decide who holds the rights to all three bundled images and publish an eligible open license or replace the images.
- [ ] Check and complete `PRIVACY.md` (controller, contact, processing, hosting and retention). Link it from the website and GitHub download page.
- [ ] Implement privacy explanation **during installation** and an option to disable applicable nonessential transfers. The current installer does not do this; documentation alone is insufficient.
- [ ] Review `CODE_SIGNING_POLICY.md`, including real authors, reviewers and approvers. Turn on MFA for GitHub and SignPath.
- [ ] Upload the documentation files alongside the source. Keep `private/`, `.env`, generated artifacts and private catalog keys out of Git.
- [ ] Verify GitHub Actions **Build Windows launcher (unsigned)** succeeds; this workflow does NOT yet sign binaries.
- [ ] Publish a genuinely usable released version with documentation and a download page. A fresh ZIP in a repo alone is not a finished release.
- [ ] Apply via https://signpath.org/ and wait for approval; approval is not automatic.
- [ ] Configure SignPath's GitHub App, trusted build pipeline and artifact/signing policies after acceptance.
- [ ] Fix the production signing order: sign the launcher; insert that exact signed file into setup payload; regenerate pinned SHA-256; build and sign setup; privately update and re-sign the update catalog based on the final launcher binary.
- [ ] Test installer, uninstall, offline behavior, privacy choices and both signed executables on a fresh Windows installation.

Official conditions: https://signpath.org/terms
GitHub integration: https://docs.signpath.io/trusted-build-systems/github
