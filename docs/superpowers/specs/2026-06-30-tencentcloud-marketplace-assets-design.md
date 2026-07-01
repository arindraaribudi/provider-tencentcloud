# Tencent Cloud Marketplace Assets — Design

**Date:** 2026-06-30
**Scope:** Add documentation, icon, and CI pipeline wiring so `provider-tencentcloud` packages show on the Upbound Marketplace with rich listing metadata.

## Context

The provider already builds and publishes an `.xpkg` to GHCR, then mirrors it to `xpkg.upbound.io/arindraaribudi/provider-tencentcloud` via `up xpkg push`. The marketplace listing currently renders with no icon and no per-version documentation. The Upbound Marketplace supports additive assets (icon, readme, docs, release notes, SBOM) appended to a published xpkg via `up alpha xpkg-append --extensions-root=./extensions`.

This change adds those assets and integrates `up alpha xpkg-append` into the existing mirror job so every release-tagged publish ships enriched marketplace content.

## Goals

- Marketplace listing shows: icon, readme, getting-started doc.
- Pipeline fully automatic on tag push and manual dispatch with `version` input.
- Existing GHCR xpkg source stays canonical and unmodified.

## Non-Goals

- Release-notes or SBOM assets (deferred — user requested docs + icon only).
- Multi-page docs (only `getting-started.md`).
- Changes to provider code, CRDs, or `crossplane.yaml` annotations (out of scope; can be follow-up).
- Altering trigger conditions or jobs other than `mirror-to-xpkg-upbound-io`.

## Repo Additions

```
extensions/
├── icons/icon.svg          ← copy from .github/03_Tcloud-icon.svg
├── readme/readme.md        ← copy from ./README.md
└── docs/getting-started.md ← new, provider-specific install + auth + minimal example
```

Source files (`README.md`, `.github/03_Tcloud-icon.svg`) stay untouched. CI reads `extensions/` directly via the `--extensions-root` flag.

## `extensions/docs/getting-started.md` Content

Single page, ~60–100 lines. Sections:

1. **Install** — `up` and `crossplane-provider` snippets pointing at `xpkg.upbound.io/arindraaribudi/provider-tencentcloud`.
2. **Authenticate** — `TENCENTCLOUD_SECRET_ID` / `TENCENTCLOUD_SECRET_KEY` (env vars on the provider pod or Kubernetes `Secret`).
3. **Configure** — minimal `ProviderConfig` example.
4. **First resource** — one CVM or CBS instance example referencing `examples-generated/`.
5. **Next steps** — pointer to `examples-generated/` for full coverage.

Skips generic Crossplane primer — README already covers it. No emoji. Uses standard Markdown only.

## CI Changes

Modify only the `mirror-to-xpkg-upbound-io` job. Trigger condition unchanged.

### Current sequence (keep + drop)

1. ✅ Keep: `crane pull "${CROSSPLANE_REGORG}/${PROVIDER_REPO}:${VERSION}" /tmp/mirror.xpkg` — for size log + sanity check.
2. ❌ Drop: final `up xpkg push "${UPBOUND_REGORG}/${PROVIDER_REPO}:${VERSION}" -f /tmp/mirror.xpkg`.

### New sequence

```yaml
- name: Resolve source digest
  env:
    VERSION: ${{ inputs.version != '' && inputs.version || github.ref_name }}
    CROSSPLANE_REGORG: ${{ env.CROSSPLANE_REGORG }}
    PROVIDER_REPO: ${{ env.PROVIDER_REPO }}
  run: |
    set -euo pipefail
    echo "DIGEST=$(crane digest "${CROSSPLANE_REGORG}/${PROVIDER_REPO}:${VERSION}")" >> "$GITHUB_ENV"

- name: Append marketplace assets
  env:
    VERSION: ${{ inputs.version != '' && inputs.version || github.ref_name }}
    CROSSPLANE_REGORG: ${{ env.CROSSPLANE_REGORG }}
    UPBOUND_REGORG: ${{ env.UPBOUND_REGORG }}
    PROVIDER_REPO: ${{ env.PROVIDER_REPO }}
    DIGEST: ${{ env.DIGEST }}
  run: |
    set -euo pipefail
    up alpha xpkg-append \
      --extensions-root=./extensions \
      --destination="${UPBOUND_REGORG}/${PROVIDER_REPO}:${VERSION}" \
      "${CROSSPLANE_REGORG}/${PROVIDER_REPO}@${DIGEST}"
```

### Behavior

- `crane digest` resolves the immutable SHA for the published tag — appends operate on content, not mutable tags.
- `--destination` writes the asset-enriched xpkg directly to the Upbound tag; GHCR source tag remains untouched.
- The `pull` step is kept only to log size and as a sanity check; the runtime append does not depend on the local file.
- Trigger still gated by `inputs.version != '' || startsWith(github.ref, 'refs/tags/v')` (the existing job-level `if:`).

## Risks

- **`up alpha xpkg-append` is alpha** (`v0.39.0+`). The CI installs `up` from `cli.upbound.io` with no pinned version — drift risk for an alpha command. Mitigation: the implementation plan will include pinning `up` to a specific version (recommend `v0.49.1` — the version the current docs page documents) in the existing `Install up CLI` step.
- **Icon size** — SVG is small; PNG alternatives rejected to avoid license/format ambiguity.
- **Readme drift** — `extensions/readme/readme.md` is a copy. If README updates, mirror step won't catch it. Documented in PR description; a follow-up could add a diff check.

## Open Questions

None.
