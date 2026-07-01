# Tencent Cloud Marketplace Assets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship an icon, readme, and getting-started doc with every published `provider-tencentcloud` xpkg by wiring `up alpha xpkg-append` into the release mirror job.

**Architecture:** Static `extensions/` tree at repo root (icons/readme/docs), copied inline because Upbound's `xpkg-append` reads the filesystem at runtime. CI append step pins the `up` CLI, resolves the immutable GHCR digest with `crane digest`, and writes the enriched package directly to the Upbound registry tag via `--destination` — replacing the old `up xpkg push` step.

**Tech Stack:** GitHub Actions, `up` CLI (v0.49.1 pinned), `crane` (already installed by the mirror job), Upbound Marketplace.

---

## File Structure

| Path | Action | Purpose |
|---|---|---|
| `extensions/icons/icon.svg` | Create (copy) | Provider icon, baked into xpkg |
| `extensions/readme/readme.md` | Create (copy) | Marketplace readme render |
| `extensions/docs/getting-started.md` | Create (new) | Provider-specific install/auth/example |
| `.github/workflows/ci.yml` | Modify (mirror-to-xpkg-upbound-io job) | Pin up, add digest step, swap push for append |

No code changes. No new Go files. Plan total ≈ 4 file operations across 5 tasks.

---

## Task 1: Create `extensions/icons/icon.svg`

**Files:**
- Create: `extensions/icons/icon.svg`

- [ ] **Step 1: Create the directory**

```bash
mkdir -p extensions/icons
```

- [ ] **Step 2: Copy the icon**

```bash
cp .github/03_Tcloud-icon.svg extensions/icons/icon.svg
```

- [ ] **Step 3: Verify the file**

```bash
file extensions/icons/icon.svg && head -1 extensions/icons/icon.svg
```

Expected output ends with: `XML 1.0 document, ASCII text` and `<?xml version="1.0" encoding="UTF-8"?>`.

- [ ] **Step 4: Commit**

```bash
git add extensions/icons/icon.svg
git commit -m "feat(extensions): add icon for marketplace listing"
```

---

## Task 2: Create `extensions/readme/readme.md`

**Files:**
- Create: `extensions/readme/readme.md`

- [ ] **Step 1: Create the directory**

```bash
mkdir -p extensions/readme
```

- [ ] **Step 2: Copy the existing README**

```bash
cp README.md extensions/readme/readme.md
```

- [ ] **Step 3: Verify**

```bash
diff -q README.md extensions/readme/readme.md && wc -l extensions/readme/readme.md
```

Expected: `Files README.md and extensions/readme/readme.md are identical` and the same line count as the root README (75 lines at time of writing).

- [ ] **Step 4: Commit**

```bash
git add extensions/readme/readme.md
git commit -m "feat(extensions): mirror README for marketplace readme slot"
```

---

## Task 3: Create `extensions/docs/getting-started.md`

**Files:**
- Create: `extensions/docs/getting-started.md`

- [ ] **Step 1: Create the directory**

```bash
mkdir -p extensions/docs
```

- [ ] **Step 2: Write the file**

```bash
cat > extensions/docs/getting-started.md <<'EOF'
# Getting Started with provider-tencentcloud

Install, authenticate, and create your first Tencent Cloud resource with the
provider.

## Install

Install the provider package on your Crossplane cluster:

```bash
kubectl apply -f - <<'YAML'
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-tencentcloud
spec:
  package: xpkg.upbound.io/arindraaribudi/provider-tencentcloud:v1.82.98
YAML
```

Or via the Upbound CLI:

```bash
up ctp provider install xpkg.upbound.io/arindraaribudi/provider-tencentcloud:v1.82.98
```

## Authentication

The provider requires a Tencent Cloud SecretId and SecretKey pair. Create a
Kubernetes Secret and reference it from a `ProviderConfig`:

```bash
kubectl create secret generic tencentcloud-creds \
  --from-literal=secret_id=$TENCENTCLOUD_SECRET_ID \
  --from-literal=secret_key=$TENCENTCLOUD_SECRET_KEY
```

```yaml
apiVersion: pkg.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: tencentcloud
spec:
  credentials:
    source: Secret
    secretRefs:
      - name: tencentcloud-creds
        key: secret_id
      - name: tencentcloud-creds
        key: secret_key
```

## First resource

Create a minimal CVM instance to verify the wiring:

```yaml
apiVersion: compute.tencentcloud.crossplane.io/v1alpha1
kind: Instance
metadata:
  name: hello-tencentcloud
spec:
  forProvider:
    instanceName: hello
    instanceType: S5.SMALL2
    imageId: img-9qrfxu9d
    availabilityZone: ap-guangzhou-3
    systemDisk:
      diskType: CLOUD_PREMIUM
      diskSize: 20
  providerConfigRef:
    name: tencentcloud
```

Apply, then watch:

```bash
kubectl get managed
```

## Next steps

See the `examples-generated/` directory for full coverage of every supported
managed resource, including VPCs, CBS volumes, RDS databases, TKE clusters, and
CLB load balancers.
EOF
```

- [ ] **Step 3: Verify**

```bash
wc -l extensions/docs/getting-started.md && head -1 extensions/docs/getting-started.md
```

Expected: roughly 70–80 lines and first line `# Getting Started with provider-tencentcloud`.

- [ ] **Step 4: Commit**

```bash
git add extensions/docs/getting-started.md
git commit -m "feat(extensions): add getting-started docs page"
```

---

## Task 4: Modify `.github/workflows/ci.yml` mirror job

**Files:**
- Modify: `.github/workflows/ci.yml` (job `mirror-to-xpkg-upbound-io`)

- [ ] **Step 1: Read the current mirror job lines**

```bash
sed -n '163,220p' .github/workflows/ci.yml
```

Expected: shows the existing `Pull xpkg artifact from GHCR` and `up xpkg push to Upbound Marketplace` steps. Use this to anchor the edits below.

- [ ] **Step 2: Pin the `up` CLI version in the install step**

Replace the body of the existing `Install up CLI` step (currently `curl -fsSL https://cli.upbound.io | sh`) with a pinned release. The new body is:

```yaml
      - name: Install up CLI (pinned)
        run: |
          set -euo pipefail
          UP_VERSION="v0.49.1"
          curl -fsSL "https://cli.upbound.io?version=${UP_VERSION}" -o up.tar
          tar -xf up.tar
          sudo mv up /usr/local/bin/
          rm up.tar
          up version
```

Match indentation exactly (6 spaces for step content, 10 for run block).

- [ ] **Step 3: Add a digest resolution step after the pull step**

Insert directly after the existing `Pull xpkg artifact from GHCR` step's closing blank line. Find the line `          ls -lh /tmp/mirror.xpkg` and add immediately after the next blank line:

```yaml
      - name: Resolve source digest
        env:
          VERSION: ${{ inputs.version != '' && inputs.version || github.ref_name }}
          CROSSPLANE_REGORG: ${{ env.CROSSPLANE_REGORG }}
          PROVIDER_REPO: ${{ env.PROVIDER_REPO }}
        run: |
          set -euo pipefail
          echo "DIGEST=$(crane digest "${CROSSPLANE_REGORG}/${PROVIDER_REPO}:${VERSION}")" >> "$GITHUB_ENV"
          echo "Resolved digest: ${DIGEST}"
```

- [ ] **Step 4: Replace the final `up xpkg push` step with `up alpha xpkg-append`**

Delete the entire `up xpkg push to Upbound Marketplace` step — from its `name:` line through its closing backticks ``` ` ```. The block to remove is exactly these 12 lines (verify with `grep -n 'up xpkg push'` then `sed -i START,ENDd .github/workflows/ci.yml` where START..END cover it):

```yaml
      - name: up xpkg push to Upbound Marketplace
        env:
          VERSION: ${{ inputs.version != '' && inputs.version || github.ref_name }}
          UPBOUND_REGORG: ${{ env.UPBOUND_REGORG }}
          PROVIDER_REPO: ${{ env.PROVIDER_REPO }}
        run: |
          set -euo pipefail
          up xpkg push "${UPBOUND_REGORG}/${PROVIDER_REPO}:${VERSION}" -f /tmp/mirror.xpkg
```

In its place, add:

```yaml
      - name: Append marketplace assets
        env:
          VERSION: ${{ inputs.version != '' && inputs.version || github.ref_name }}
          UPBOUND_REGORG: ${{ env.UPBOUND_REGORG }}
          CROSSPLANE_REGORG: ${{ env.CROSSPLANE_REGORG }}
          PROVIDER_REPO: ${{ env.PROVIDER_REPO }}
          DIGEST: ${{ env.DIGEST }}
        run: |
          set -euo pipefail
          up alpha xpkg-append \
            --extensions-root=./extensions \
            --destination="${UPBOUND_REGORG}/${PROVIDER_REPO}:${VERSION}" \
            "${CROSSPLANE_REGORG}/${PROVIDER_REPO}@${DIGEST}"
```

- [ ] **Step 5: Validate the workflow YAML**

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml')); print('OK')"
```

Expected: prints `OK`. If it throws `yaml.scanner.ScannerError`, fix indentation to match the existing steps.

- [ ] **Step 6: Confirm mirror job is the only job touched**

```bash
grep -n 'job:' .github/workflows/ci.yml
```

Expected: 5 jobs listed (`detect-noop`, `lint`, `check-diff`, `unit-tests`, `publish-artifacts`, `mirror-to-xpkg-upbound-io`). Only `mirror-to-xpkg-upbound-io` should contain the new `xpkg-append` call.

- [ ] **Step 7: Confirm no `up xpkg push` call remains**

```bash
grep -n 'up xpkg push' .github/workflows/ci.yml || echo "no matches (expected)"
```

Expected: `no matches (expected)`.

- [ ] **Step 8: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "feat(ci): append marketplace assets (icon, readme, docs) on release"
```

---

## Task 5: Validate locally and push

**Files:**
- No new files

- [ ] **Step 1: Sanity-check the extensions tree**

```bash
find extensions -type f | sort
```

Expected: exactly three files:

```
extensions/docs/getting-started.md
extensions/icons/icon.svg
extensions/readme/readme.md
```

- [ ] **Step 2: Confirm CI YAML still parses**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml')); print('OK')"
```

Expected: `OK`.

- [ ] **Step 3: Push the branch and open PR**

```bash
git push -u origin HEAD
gh pr create \
  --title "ci: append marketplace assets on release" \
  --body "$(cat <<'EOF'
## Summary
- Add `extensions/` tree (icon, readme, getting-started doc) for Upbound Marketplace rendering.
- Pin `up` CLI to `v0.49.1` in the mirror job.
- Replace `up xpkg push` step with `up alpha xpkg-append --destination=...` so each release publishes with icon + readme + docs.

## Test plan
- [ ] Workflow YAML validates
- [ ] Manual: tag a test release and verify marketplace listing renders icon + readme + docs
- [ ] Confirm GHCR source tag is unmodified

Refs: docs/superpowers/specs/2026-06-30-tencentcloud-marketplace-assets-design.md

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 4: Verify PR exists and CI is running**

```bash
gh pr view --json number,url,statusCheckRollup
```

Expected: returns a JSON object with `url` pointing to the new PR and `statusCheckRollup` showing the workflows dispatched.

---

## Notes for the Executor

- **No Go code, no tests** — this is a docs + CI change. The YAML parse step (Task 4.5) and the `find` step (Task 5.1) are the closest thing to automated verification.
- **Readme drift**: `extensions/readme/readme.md` is a snapshot. If the root README changes, re-copy. A future task could automate this with a `make` target.
- **Alpha feature**: `up alpha xpkg-append` is documented as alpha. The pin (`v0.49.1`) minimizes drift; if the command signature changes in a future release, bump the pin and audit.
- **Not addressed by this plan** (per spec non-goals): `crossplane.yaml` annotations (license, source URL, maintainer). Follow-up task.
