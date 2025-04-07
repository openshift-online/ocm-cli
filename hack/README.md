# OCM Helper Scripts

These scripts help in the process of releasing new versions of `ocm`.

Once all changes for a specific release are in `main`, the next step is to
create a release commit:

	./hack/commit-release.sh

This creates a new branch, updates the OCM build version and changelog file
then commits and pushes to GitHub. Any potentially destructing action has a
confirmation prompt.

Once this new branch is pushed, someone has to merge it. Once merged, make sure
to update your local copy. Then you can tag the actual release:

	./hack/tag-release.sh

This will create a new annotated tag and push it to the upstream OCM
repository.

Now that the tag is in place, you will go to the
[tags page](https://github.com/openshift/ocm/tags) and edit the latest one. In
there make sure that the release title and description match the release tag
annotation.

Publish the release and you're done.

# Konflux Release
