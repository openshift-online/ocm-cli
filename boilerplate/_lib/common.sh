err() {
  echo "==ERROR== $@" >&2
  exit 1
}

banner() {
    echo
    echo "=============================="
    echo "$@"
    echo "=============================="
}

## osdk_version BINARY
#
# Print the version of the specified operator-sdk BINARY
osdk_version() {
    local osdk=$1
    # `operator-sdk version` output looks like
    #       operator-sdk version: v0.8.2, commit: 28bd2b0d4fd25aa68e15d928ae09d3c18c3b51da
    # or
    #       operator-sdk version: "v0.16.0", commit: "55f1446c5f472e7d8e308dcdf36d0d7fc44fc4fd", go version: "go1.13.8 linux/amd64"
    # Peel out the version number, accounting for the optional quotes.
    $osdk version | ${SED?} 's/operator-sdk version: "*\([^,"]*\)"*,.*/\1/'
}

## opm_version BINARY
#
# Print the version of the specified opm BINARY
opm_version() {
    local opm=$1
    # `opm version` output looks like:
    #    Version: version.Version{OpmVersion:"v1.15.2", GitCommit:"fded0bf", BuildDate:"2020-11-18T14:21:24Z", GoOs:"darwin", GoArch:"amd64"}
    $opm version | ${SED?} 's/.*OpmVersion:"//;s/".*//'
}

## grpcurl_version BINARY
#
# Print the version of the specified grpcurl BINARY
grpcurl_version() {
    local grpcurl=$1
    # `grpcurl -version` output looks like:  grpcurl 1.7.0
    $grpcurl -version 2>&1 | cut -d " " -f 2
}

## repo_import REPODIR
#
# Print the qualified org/name of the current repository, e.g.
# "openshift/wizbang-foo-operator". This relies on git remotes being set
# reasonably.
repo_name() {
    # Just strip off the first component of the import-ish path
    repo_import $1 | ${SED?} 's,^[^/]*/,,'
}

## repo_import REPODIR
#
# Print the go import-ish path to the current repository, e.g.
# "github.com/openshift/wizbang-foo-operator". This relies on git
# remotes being set reasonably.
repo_import() {
    # Account for remotes which are
    # - upstream or origin
    # - ssh ("git@host.com:org/name.git") or https ("https://host.com/org/name.git")
    (git -C $1 config --get remote.upstream.url || git -C $1 config --get remote.origin.url) | ${SED?} 's,git@\([^:]*\):,\1/,; s,https://,,; s/\.git$//'
}

## current_branch REPO
#
# Outputs the name of the current branch in the REPO directory
current_branch() {
    (
        cd $1
        git rev-parse --abbrev-ref HEAD
    )
}

## image_exists_in_repo IMAGE_URI
#
# Checks whether IMAGE_URI -- e.g. quay.io/app-sre/osd-metrics-exporter:abcd123
# -- exists in the remote repository.
# If so, returns success.
# If the image does not exist, but the query was otherwise successful, returns
# failure.
# If the query fails for any reason, prints an error and *exits* nonzero.
image_exists_in_repo() {
    local image_uri=$1
    local output
    local rc

    local skopeo_stderr=$(mktemp)

    if ! command -v skopeo &>/dev/null; then
      echo "Failed to find the skopeo binary. If you are on Mac: brew install skopeo." >&2
      exit 1
    fi

    output=$(skopeo inspect docker://${image_uri} 2>$skopeo_stderr)
    rc=$?
    # So we can delete the temp file right away...
    stderr=$(cat $skopeo_stderr)
    rm -f $skopeo_stderr
    if [[ $rc -eq 0 ]]; then
        # The image exists. Sanity check the output.
        local digest=$(echo $output | jq -r .Digest)
        if [[ -z "$digest" ]]; then
            echo "Unexpected error: skopeo inspect succeeded, but output contained no .Digest"
            echo "Here's the output:"
            echo "$output"
            echo "...and stderr:"
            echo "$stderr"
            exit 1
        fi
        echo "Image ${image_uri} exists with digest $digest."
        return 0
    elif [[ "$output" == *"manifest unknown"* || "$stderr" == *"manifest unknown"* ]]; then
        # We were able to talk to the repository, but the tag doesn't exist.
        # This is the normal "green field" case.
        echo "Image ${image_uri} does not exist in the repository."
        return 1
    elif [[ "$output" == *"manifest unknown"* || "$stderr" == *"was deleted or has expired"* ]]; then
        # This should be rare, but accounts for cases where we had to
        # manually delete an image.
        echo "Image ${image_uri} was deleted from the repository."
        echo "Proceeding as if it never existed."
        return 1
    else
        # Any other error. For example:
        #   - "unauthorized: access to the requested resource is not
        #     authorized". This happens not just on auth errors, but if we
        #     reference a repository that doesn't exist.
        #   - "no such host".
        #   - Network or other infrastructure failures.
        # In all these cases, we want to bail, because we don't know whether
        # the image exists (and we'd likely fail to push it anyway).
        echo "Error querying the repository for ${image_uri}:"
        echo "stdout: $output"
        echo "stderr: $stderr"
        exit 1
    fi
}

if [ "$BOILERPLATE_SET_X" ]; then
  set -x
fi

# Only used for error messages
_lib=${BASH_SOURCE##*/}

# When this lib is sourced (which is what it's designed for), $0 is the
# script that did the sourcing.
SOURCER=$(realpath $0)
[[ -n "$SOURCER" ]] || err "$_lib couldn't discover where it was sourced from"

HERE=${SOURCER%/*}
[[ -n "$HERE" ]] || err "$_lib failed to discover the dirname of sourcer at $SOURCER"

REPO_ROOT=$(git rev-parse --show-toplevel)
[[ -n "$REPO_ROOT" ]] || err "$_lib couldn't discover the repo root"

CONVENTION_ROOT=$REPO_ROOT/boilerplate
[[ -d "$CONVENTION_ROOT" ]] || err "$CONVENTION_ROOT: not a directory"

# Were we sourced from within a convention?
if [[ "$HERE" == "$CONVENTION_ROOT/"* ]]; then
  # Okay, figure out the name of the convention
  CONVENTION_NAME=${HERE#$CONVENTION_ROOT/}
  # If we got here, we really expected to be able to identify the
  # convention name.
  [[ -n "$CONVENTION_NAME" ]] || err "$_lib couldn't discover the name of the sourcing convention"
fi

# Set SED variable
if LANG=C sed --help 2>&1 | grep -q GNU; then
  SED="sed"
elif command -v gsed &>/dev/null; then
  SED="gsed"
else
  echo "Failed to find GNU sed as sed or gsed. If you are on Mac: brew install gnu-sed." >&2
  exit 1
fi

if [ -z "$BOILERPLATE_GIT_REPO" ]; then
  export BOILERPLATE_GIT_REPO=https://github.com/openshift/boilerplate.git
fi

# Base image repo url
IMAGE_REPO=quay.io/redhat-services-prod/openshift
# The namespace of the ImageStream by which prow will import the image.
IMAGE_NAMESPACE=openshift
IMAGE_NAME=boilerplate
# LATEST_IMAGE_TAG may be set manually or by `update`, in which case
# that's the value we want to use.
if [[ -z "$LATEST_IMAGE_TAG" && -z "$SKIP_LATEST_IMAGE_TAG_RESOLUTION" ]]; then
    # (Non-ancient) consumers will have the tag in this file.
    if [[ -f ${CONVENTION_ROOT}/_data/backing-image-tag ]]; then
        LATEST_IMAGE_TAG=$(cat ${CONVENTION_ROOT}/_data/backing-image-tag)

    # In boilerplate itself, we can discover the latest from git.
    elif [[ $(repo_name .) == openshift/boilerplate ]]; then
        LATEST_IMAGE_TAG=$(git describe --tags --abbrev=0 --match image-v*)
    fi
fi
# The public image location
IMAGE_PULL_PATH=${IMAGE_PULL_PATH:-$IMAGE_REPO/$IMAGE_NAME:$LATEST_IMAGE_TAG}
