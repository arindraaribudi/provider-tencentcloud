#!/usr/bin/env bash
set -euo pipefail

# Resolve project root regardless of caller cwd.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

# ── Configurable variables ─────────────────────────────────────────────────
REGISTRY="${REGISTRY:-docker.io/marve39}"
IMAGE_NAME="${IMAGE_NAME:-provider-tencentcloud}"
TAG="${TAG:-$(git describe --tags --always --dirty 2>/dev/null || echo latest)}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"

TERRAFORM_VERSION="${TERRAFORM_VERSION:-1.2.1}"
TERRAFORM_PROVIDER_SOURCE="${TERRAFORM_PROVIDER_SOURCE:-tencentcloudstack/tencentcloud}"
TERRAFORM_PROVIDER_VERSION="${TERRAFORM_PROVIDER_VERSION:-1.82.98}"
TERRAFORM_PROVIDER_DOWNLOAD_NAME="${TERRAFORM_PROVIDER_DOWNLOAD_NAME:-terraform-provider-tencentcloud}"
TERRAFORM_PROVIDER_DOWNLOAD_URL_PREFIX="${TERRAFORM_PROVIDER_DOWNLOAD_URL_PREFIX:-https://releases.hashicorp.com/${TERRAFORM_PROVIDER_DOWNLOAD_NAME}/${TERRAFORM_PROVIDER_VERSION}}"
TERRAFORM_NATIVE_PROVIDER_BINARY="${TERRAFORM_NATIVE_PROVIDER_BINARY:-${TERRAFORM_PROVIDER_DOWNLOAD_NAME}_v${TERRAFORM_PROVIDER_VERSION}}"

CONTROLLER_IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}-controller"
PACKAGE_IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"
IMAGE_DIR="${PROJECT_ROOT}/cluster/images/provider-tencentcloud"
DOCKERFILE="${IMAGE_DIR}/Dockerfile"
PACKAGE_DIR="${PROJECT_ROOT}/package"
PACKAGE_YAML="${PACKAGE_DIR}/package.yaml"
OUTPUT_XPKG="${PROJECT_ROOT}/${IMAGE_NAME}-${TAG}.xpkg"

# ── 1. Build provider binaries for each platform ───────────────────────────
echo "▶ Building provider binaries"
IFS=',' read -ra PLATFORM_LIST <<< "${PLATFORMS}"
for PLATFORM in "${PLATFORM_LIST[@]}"; do
  TARGETOS="${PLATFORM%%/*}"
  TARGETARCH="${PLATFORM##*/}"
  BIN_OUT="${PROJECT_ROOT}/bin/${TARGETOS}_${TARGETARCH}"
  echo "  → ${TARGETOS}/${TARGETARCH}"
  mkdir -p "${BIN_OUT}"
  CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
    go build -o "${BIN_OUT}/provider" ./cmd/provider
  # Stage into image dir for Docker build context
  mkdir -p "${IMAGE_DIR}/bin/${TARGETOS}_${TARGETARCH}"
  cp "${BIN_OUT}/provider" "${IMAGE_DIR}/bin/${TARGETOS}_${TARGETARCH}/provider"
done

# ── 2. Build multi-platform controller image ───────────────────────────────
# Multi-arch manifests cannot be loaded locally; must push directly.
echo "▶ Building multi-platform controller image → ${CONTROLLER_IMAGE}"

BUILDX_PUSH_FLAG=""
if [[ "${PUSH:-0}" == "1" ]]; then
  BUILDX_PUSH_FLAG="--push"
else
  # Without push, load single-platform image for local testing
  FIRST_PLATFORM="${PLATFORM_LIST[0]}"
  echo "  (PUSH=0: building ${FIRST_PLATFORM} only, loading locally)"
  PLATFORMS="${FIRST_PLATFORM}"
  BUILDX_PUSH_FLAG="--load"
fi

docker buildx build \
  --platform "${PLATFORMS}" \
  --build-arg TERRAFORM_VERSION="${TERRAFORM_VERSION}" \
  --build-arg TERRAFORM_PROVIDER_SOURCE="${TERRAFORM_PROVIDER_SOURCE}" \
  --build-arg TERRAFORM_PROVIDER_VERSION="${TERRAFORM_PROVIDER_VERSION}" \
  --build-arg TERRAFORM_PROVIDER_DOWNLOAD_NAME="${TERRAFORM_PROVIDER_DOWNLOAD_NAME}" \
  --build-arg TERRAFORM_NATIVE_PROVIDER_BINARY="${TERRAFORM_NATIVE_PROVIDER_BINARY}" \
  --build-arg TERRAFORM_PROVIDER_DOWNLOAD_URL_PREFIX="${TERRAFORM_PROVIDER_DOWNLOAD_URL_PREFIX}" \
  -f "${DOCKERFILE}" \
  -t "${CONTROLLER_IMAGE}" \
  ${BUILDX_PUSH_FLAG} \
  "${IMAGE_DIR}"

rm -rf "${IMAGE_DIR:?}/bin"

# ── 3. Update package.yaml controller image ────────────────────────────────
echo "▶ Updating ${PACKAGE_YAML} → image: ${CONTROLLER_IMAGE}"
sed -i.bak "s|image:.*|image: ${CONTROLLER_IMAGE}|" "${PACKAGE_YAML}" && rm -f "${PACKAGE_YAML}.bak"

# ── 4. Build Crossplane xpkg ───────────────────────────────────────────────
# xpkg must embed the amd64 controller — pull by digest to avoid Apple Silicon
# resolving the multi-arch tag to arm64 locally.
echo "▶ Pulling amd64 controller image for xpkg base"
AMD64_DIGEST=$(docker buildx imagetools inspect "${CONTROLLER_IMAGE}" --raw \
  | python3 -c "import json,sys; m=json.load(sys.stdin); \
    print(next(x['digest'] for x in m.get('manifests',[]) \
    if x.get('platform',{}).get('architecture')=='amd64' \
    and x.get('platform',{}).get('os')=='linux'))")
CONTROLLER_AMD64_REF="${REGISTRY}/${IMAGE_NAME}@${AMD64_DIGEST}"
DOCKER_DEFAULT_PLATFORM=linux/amd64 docker pull "${CONTROLLER_AMD64_REF}"

echo "▶ Building xpkg → ${OUTPUT_XPKG}"
up xpkg build \
  --package-root "${PACKAGE_DIR}" \
  --controller "${CONTROLLER_AMD64_REF}" \
  --output "${OUTPUT_XPKG}"

if [[ "${PUSH:-0}" == "1" ]]; then
  echo "▶ Pushing xpkg → ${PACKAGE_IMAGE}"
  up xpkg push \
    --package "${OUTPUT_XPKG}" \
    "${PACKAGE_IMAGE}"
fi

echo "✓ Done"
echo "  Controller image : ${CONTROLLER_IMAGE}  (linux/amd64, linux/arm64)"
echo "  Package image    : ${PACKAGE_IMAGE}"
echo "  Xpkg             : ${OUTPUT_XPKG}"