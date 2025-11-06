#!/bin/bash
#
# build_image.sh - Build a standard container image for OCM CLI
#
# This script builds a container image using podman with network access enabled.
# Unlike build_hermetic_image.sh, this allows network access during the build
# process, enabling dependency downloads and standard build workflows.
#
# Usage:
#   ./build_image.sh <IMAGE_REPOSITORY> <IMAGE_TAG> <IMAGE_NAME>
#
# Arguments:
#   IMAGE_REPOSITORY - Container registry repository (e.g., quay.io/openshift)
#   IMAGE_TAG        - Tag for the image (e.g., v1.0.8, latest)
#   IMAGE_NAME       - Name of the image (e.g., ocm-cli)
#
# Example:
#   ./build_image.sh quay.io/openshift v1.0.8 ocm-cli
#   # Results in: quay.io/openshift/ocm-cli:v1.0.8
#
# Prerequisites:
#   - podman must be installed and available in PATH
#   - docker/Dockerfile must exist in the project root
#   - Network access for downloading dependencies during build
#
# Build characteristics:
#   - Uses --no-cache to ensure fresh build
#   - Allows network access for dependency downloads
#   - Standard (non-hermetic) build process
#   - Suitable for development and testing builds
#
# Note: For reproducible/hermetic builds, use build_hermetic_image.sh instead
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

# Build the standard container image with network access
echo "Creating image $IMAGE_REPOSITORY/$IMAGE_NAME:$IMAGE_TAG..."
podman build . \
  --file docker/Dockerfile \
  --no-cache \
  --tag $IMAGE_REPOSITORY/$IMAGE_NAME:$IMAGE_TAG
