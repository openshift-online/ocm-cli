#!/bin/bash
#
# build_hermetic_image.sh - Build a hermetically-sealed container image for OCM CLI
#
# This script builds a container image using podman with network isolation (hermetic build)
# to ensure reproducible builds. The build process uses pre-fetched dependencies from
# hermeto-output/ and environment configuration from hermeto.env.
#
# Usage:
#   ./build_hermetic_image.sh <IMAGE_REPOSITORY> <IMAGE_TAG> <IMAGE_NAME>
#
# Arguments:
#   IMAGE_REPOSITORY - Container registry repository (e.g., quay.io/openshift)
#   IMAGE_TAG        - Tag for the image (e.g., v1.0.8, latest)
#   IMAGE_NAME       - Name of the image (e.g., ocm-cli)
#
# Example:
#   ./build_hermetic_image.sh quay.io/openshift v1.0.8 ocm-cli
#   # Results in: quay.io/openshift/ocm-cli:v1.0.8
#
# Prerequisites:
#   - podman must be installed and available in PATH
#   - ./hermeto-output/ directory must exist with pre-fetched dependencies
#   - ./hermeto.env file must exist with environment configuration
#   - docker/Dockerfile must exist in the project root
#
# Build characteristics:
#   - Uses --no-cache to ensure fresh build
#   - Uses --network none for hermetic (network-isolated) build
#   - Mounts hermeto-output/ and hermeto.env as read-only volumes
#   - SELinux compatibility with :Z volume mount option
#

# Validate required arguments
MISSING_PARAM=false;
if [ -z "$1" ]; then
  echo "Error: IMAGE_REPOSITORY argument cannot be empty"
  MISSING_PARAM=true;
fi

if [ -z "$2" ]; then
  echo "Error: IMAGE_TAG argument cannot be empty"
  MISSING_PARAM=true;
fi

if [ -z "$3" ]; then
  echo "Error: IMAGE_NAME argument cannot be empty"
  MISSING_PARAM=true;
fi

# Exit if any required parameters are missing
if $MISSING_PARAM; then
    exit 1
fi

# Store validated arguments in descriptive variables
IMAGE_REPOSITORY=$1
IMAGE_TAG=$2
IMAGE_NAME=$3

TMP_DIR=`mktemp -d`

function die() {
    echo 'ERROR:' $1
    exit 1
}

hermeto --version &> /dev/null || die 'hermeto not installed'

echo "Fetching dependencies ($TMP_DIR/fetch_deps.log)..."
hermeto fetch-deps \
  --source ./ \
  --output $TMP_DIR/hermeto-output \
  --sbom-output-type cyclonedx \
  '{"path": ".", "type": "gomod"}' &> $TMP_DIR/fetch_deps.log || die

echo "Generating hermetic environment ($TMP_DIR/generate_env.log)..."
hermeto generate-env \
    $TMP_DIR/hermeto-output \
    -o $TMP_DIR/hermeto.env \
    --for-output-dir /tmp/hermeto-output &> $TMP_DIR/generate_env.log || die


echo "Injecting files ($TMP_DIR/inject_files.log)..."
hermeto inject-files \
    $TMP_DIR/hermeto-output \
    --for-output-dir /tmp/hermeto-output &> $TMP_DIR/inject_files.log || die

# The chmod 777 is necessary because:
# 1. The temp directory is created with host user permissions
# 2. Inside the container, the build process runs as a different user
# 3. Without open permissions, the container can't access the mounted Go module cache
# 4. This is safe because it's a temporary directory that gets cleaned up after the build
chmod -R 777 $TMP_DIR/hermeto-output/deps/gomod

# Build the hermetically-sealed container image
echo "Creating hermetically-sealed image $IMAGE_REPOSITORY/$IMAGE_NAME:$IMAGE_TAG..."
podman build . \
  --file docker/Dockerfile \
  --no-cache \
  --volume "$TMP_DIR/hermeto-output":/tmp/hermeto-output:Z \
  --volume "$TMP_DIR/hermeto.env":/tmp/hermeto.env:Z \
  --network none \
  --tag $IMAGE_REPOSITORY/$IMAGE_NAME:$IMAGE_TAG | tee $TMP_DIR/build_image.log
