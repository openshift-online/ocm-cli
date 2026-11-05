# Helpers and variables for dealing with openshift/release

# NOTE: This library is sourced from user-run scripts. It should not be
# sourced in CI, as it relies on git config that's not necessarily
# present there.

RELEASE_REPO=openshift/release

## Information about the boilerplate consumer
# E.g. "openshift/my-wizbang-operator"
CONSUMER=$(repo_name .)
[[ -z "$CONSUMER" ]] && err "
Failed to determine current repository name"
#
# E.g. "openshift"
CONSUMER_ORG=${CONSUMER%/*}
[[ -z "$CONSUMER_ORG" ]] && err "
Failed to determine consumer org"
#
# E.g. "my-wizbang-operator"
CONSUMER_NAME=${CONSUMER#*/}
[[ -z "$CONSUMER_NAME" ]] && err "
Failed to determine consumer name"
#
# E.g. "master"
# This will produce something like refs/remotes/origin/master
DEFAULT_BRANCH=$(git symbolic-ref refs/remotes/upstream/HEAD 2>/dev/null || git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null || echo defaulting/to/master)
# Strip off refs/remotes/{upstream|origin}/
DEFAULT_BRANCH=${DEFAULT_BRANCH##*/}
[[ -z "$DEFAULT_BRANCH" ]] && err "
Failed to determine default branch name"

## release_process_args "$@"
#
# This is for use by commands expecting one optional argument which is
# the file system path to a clone of the $RELEASE_REPO.
#
# Will invoke `usage` -- which must be defined by the caller -- if
# the wrong number of arguments are received, or if the single argument
# is `help` or a flag.
#
# If exactly one argument is specified and it is valid, it is assigned
# to the global RELEASE_CLONE variable.
release_process_args() {
    if [[ $# -eq 1 ]]; then
        # Special cases for usage queries
        if [[ "$1" == '-'* ]] || [[ "$1" == help ]]; then
            usage
        fi

        [[ -d $1 ]] || err "
    $1: Not a directory."

        [[ $(repo_name $1) == "$RELEASE_REPO" ]] || err "
    $1 is not a clone of $RELEASE_REPO; or its 'origin' remote is not set properly."

        # Got a usable clone of openshift/release
        RELEASE_CLONE="$1"

    elif [[ $# -ne 0 ]]; then
        usage
    fi
}

## release_validate_invocation
#
# Make sure we were called from a reasonable place, that being:
# - A boilerplate consumer
# - ...that's actually subscribed to a convention
# - ...containing the script being invoked
release_validate_invocation() {
    # Make sure we were invoked from a boilerplate consumer.
    [[ -z "$CONVENTION_NAME" ]] && err "
    $cmd must be invoked from a consumer of an appropriate convention. Where did you get this script from?"
    # Or at least not from boilerplate itself
    [[ "$CONSUMER" == "openshift/boilerplate" ]] && err "
    $cmd must be invoked from a boilerplate consumer, not from boilerplate itself."

    [[ -s $CONVENTION_ROOT/_data/last-boilerplate-commit ]] || err "
    $cmd must be invoked from a boilerplate consumer!"

    grep -E -q "^$CONVENTION_NAME(\s.*)?$" $CONVENTION_ROOT/update.cfg || err "
    $CONSUMER is not subscribed to $CONVENTION_NAME!"
}

## release_prep_clone
#
# If $RELEASE_CLONE is already set:
# - It should represent a directory containing a clean checkout of the
#   release repository; otherwise we error.
# - We checkout and pull master.
# Otherwise:
# - We clone the release repo to a temporary directory.
# - We set the $RELEASE_CLONE global variable to point to that
#   directory.
release_prep_clone() {
    # If a release repo clone wasn't specified, create one
    if [[ -z "$RELEASE_CLONE" ]]; then
        RELEASE_CLONE=$(mktemp -dt openshift_release_XXXXXXX)
        git clone --depth=1 git@github.com:${RELEASE_REPO}.git $RELEASE_CLONE
    else
        [[ -z "$(git -C $RELEASE_CLONE status --porcelain)" ]] || err "
Your release clone must start clean."
        # These will blow up if it's misconfigured
        git -C $RELEASE_CLONE checkout master
        git -C $RELEASE_CLONE pull
    fi
}

## release_done_msg BRANCH
#
# Print exit instructions for submitting the release PR.
# BRANCH is a suggested branch name.
release_done_msg() {
    echo
    git status

    cat <<EOF

Ready to commit, push, and create a PR in $RELEASE_CLONE
You may wish to:

cd $RELEASE_CLONE
git checkout -b $release_branch
git add -A
git commit
git push origin $release_branch
EOF
}
