## Releasing a new OCM CLI Version

Releasing a new version requires submitting an MR for review/merge with an
update to the `Version` constant in [pkg/info/info.go](pkg/info/info.go).
Additionally, update the [CHANGES.md](CHANGES.md) file to include the new
version and describe all changes included.

Below is an example CHANGES.md update:

```
## 0.1.36 Feb 14 2020

- Add `state` to list of default columns for cluster list.
- Preserve order of attributes in JSON output.
```

Submit an MR for review/merge with the CHANGES.md and Makefile update.

Finally, create and submit a new tag with the new version following the below
example:

```
git checkout main
git pull
git tag -a -m 'Release 0.1.38' v0.1.38
git push origin v0.1.38
```

Note that a repository administrator may need to push the tag to the repository
due to access restrictions.

After submitting a tag, a release will be automatically published to the
releases page by the [release action](.github/workflows/publish-release.yaml),
including the binaries for the supported platforms.

## Boilerplate

This repository subscribes to the `openshift/rosa-services` boilerplate convention in
[`boilerplate/update.cfg`](boilerplate/update.cfg). It supplies the shared root
`OWNERS_ALIASES`; this repository maintains its own `OWNERS` files, which can
reference those aliases. See the [convention README](boilerplate/openshift/rosa-services/README.md)
for details.

To pull in changes to the convention (including alias membership), run
`make boilerplate-update` from a clean checkout, review the resulting changes,
and commit them. Update shared aliases in the upstream
[openshift/boilerplate](https://github.com/openshift/boilerplate) repository rather
than editing the copied `OWNERS_ALIASES` or managed boilerplate files here.
