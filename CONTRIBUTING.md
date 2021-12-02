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
git checkout master
git pull
git tag -a -m 'Release 0.1.38' v0.1.38
git push origin v0.1.38
```

Note that a repository administrator may need to push the tag to the repository
due to access restrictions.

After submitting a tag, a release will be automatically published to the
releases page by the [release action](.github/workflows/publish-release.yaml),
including the binaries for the supported platforms.
