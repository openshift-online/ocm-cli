# Changes

This document describes the relevant changes between releases of the
`ocm` command line tool.

## 0.1.64 Jul 6 2022

- Add extra scopes support for OpenID IDP
- config: Allow using encrypted refresh tokens
- Correct the doc URLs for IDP creation
- added limited support reasons to the url_expander pkg
- better error when gcp ccs creds are not provided

## 0.1.63 May 12 2022

- Improve existing VPC and proxy code.
- Disable color in Windows.
- Added account number to "ocm describe cluster ...".
- Show limited support flag in "ocm describe cluster ...".

## 0.1.62 Feb 23 2022

- Update to SDK 0.1.240
- Only show relevant regions in interactive cluster create
- Update linter config
- Add linting GH action
- Fix linting issues
- Run go mod tidy
- Unify PR actions
- ocm-cli - Improve the cluster-wide-proxy use cases
- It should fail to create a cluster with proxy atrribute but no --use-existing-vpc
- [WIP] ocm-cli should support byovpc and cluster-wide-proxy for gcp
- It should be failed when create a non-ccs cluster with proxy attributes
- Support byovpc and cluster-wide-proxy features
- GCPNetwork attributes should show in cluster describe
- Fix HTML escaping
- It should be successful to edit a ccs cluster with proxy attribute
- Update SDK to v0.1.242

## 0.1.61 Feb 1 2022

- Add STS to `describe cluster` output.
- Add BYOPC flag to `describe cluster` command.
- Support BYO-VPC cluster creation flags.
- Support cluster-wide proxy during cluster creation and update.
- Don't allow additional trust bundle as parameter.
- Add more URLs to expander.
- Readable error message when creating a CCS cluster with invalid
  `--htts-proxy` value.

## 0.1.60 Dec 3 2021

This version doesn't contain changes to the functionality, only to the
development and build workflows:

- Rename `master` branch to `main`.

  To adapt your local repository to the new branch name run the following
  commands:

  ```shell
  git branch -m master main
  git fetch origin
  git branch -u origin/main main
  git remote set-head origin -a
  ```

- Automatically add changes from `CHANGES.md` to release descriptions.

## 0.1.59 Oct 28 2021

- Fix binary names inside `.sha256` files
- Build for all RHEL architectures
- Replace quota_summary by quota_cost
- Add alias `-p` to `--parameter`

## 0.1.58 Sep 23 2021

The only change in this relase is that the _GitHub_ action that publishes
releases has been fixed so that it publishes correct binaries. There are no
changes in functionality. See [#319](https://github.com/openshift-online/ocm-cli/issues/319)
for details.

## 0.1.57 Sep 22 2021

- Replace `go-bindata` with Go 1.16 `embed.FS`. This has no practical
  implication for users, but for developers it means that the project requires
  Go 1.16.

- Run tests using _GitHub_ actions instead of _Jenkins_. This increases the
  platform coverate as tests now run in _Linux_, _MacOS_ and _Windows_.

- Color output is now generated internally without requiring the installation
  of the `jq` tool.

- Show provisioning error code and message.

## 0.1.56 Aug 25 2021

- Use standard XDG configuration path for `ocm.json`.

  If the legacy `~/.ocm.json` file already exists it continues using that,
  otherwise it prefers the standard XDG configuration directory. That usually
  means `~/.config/ocm/ocm.json`.

  We recommend removing the old file and running the `login` command again:

  ```shell
  $ rm ~/.ocm.json
  $ ocm login ...
  ```

  Or move the existing file to the new location:

  ```shell
  $ mkdir -p ~/.config/ocm
  $ mv ~/.ocm.json ~/.config/ocm/ocm.json
  ```

- User friendly message when offline token is no longer valid.

## 0.1.55 Jul 30 2021

- Add CLI tests
- Don't remove config file on logout
- Add `pager` configuration setting
- Use pager command for `list clusters`
- Add table printer
- Add printer table for organizations
- Add table for plugins
- Add table for `addons`
- Use table for `list idps`
- Add column width learning
- Support specifying IdD Name
- feat(login): allow for empty OCM_CONFIG
- llokup clusters by subscriptions
- Update login URL for upcoming move to console.redhat.com
- Add password generation option for IDP
- Add htapssed IDP

## 0.1.54 Jun 23 2021

- Update to OCM SDK 0.1.190.
  - Don't require refresh token for client credentials grant.
  - Don't use refresh token if have client credentials.

## 0.1.53 Jun 22 2021

The only change in this release is the removal of the paging feature that was
introduced in the previous release. Users have complained that it disrupts
their workflows. In particular the fact that _less_ clears the screen after
finishing when the results fit in one page.

Note that in version of less included in many _Linux_ operating systems can be
configured to disable this screen clearing adding the `-F` option to the `LESS`
environment variable:

```shell
export LESS="-F"
```

But apparently other operating systems, in particular _macOS_, don't have this
version or less, or have a version that doesn't support that option or
environment variable.

This feature will be reintroduced later with a mechanism to persistently enable
or disable it.

## 0.1.52 Jun 20 2021

- Update ocm-sdk-go to 0.1.186
- Use `less` to page cluster list
- Added ccs_only to cloud regions
- Honor machine_types ccs_only field
- ocm post: pass correct info to ApplyHeaderFlag()
- Add option to encrypt etcd during cluster installation
- list versions by channel group
- Modify resource_name comparison when populating add-ons
- Add PrivateLink To Describe Cluster

## 0.1.51 May 25 2021

- Remove ResourceQuota Allowed field

## 0.1.50 May 20 2021

- Merge value of `--parameter search="..."` with search query generated by the
  `list clusters` command.

## 0.1.49 May 4 2021
- Update ocm-sdk-go to 0.1.173
- Commands for Job Queues
- Convert ocm account quota to ocm list quota command
- Support creating clusters in different channel groups
- Edit cluster channel group
- Adjust column padding for `list clusters`
- Add flag to suppress printing of headers
- Add cluster state to `describe cluster` command
- Allow autoscaling non-default machine pool with 0 replicas

## 0.1.48 Mar 10 2021
- Add support for hibernate / resume cluster.
- Add flag to sshuttle.
- Fix cluster admin enabled output.
- Fix empty edit default machine pool.

## 0.1.47 Feb 2 2021

- Update ocm-sdk-go to 0.1.152
- Avoid `survey.Select` bug when Default is not one of Options
- Fix CheckOneOf() error message
- Drop default of --region
- `ocm list machinepools` - added autoscaling field, and range
- `ocm create cluster` - added autoscaling params
- `ocm describe cluster` - added autoscaling indication and range
- `ocm edit machinepool` - can now edit default machine pool - and autoscaling params
- `ocm edit cluster` - no longer able to edit compute nodes

## 0.1.46 Jan 10 2021

- Show sorted version list in `ocm list versions`
- Fixed API endpoint in the README file
- Support creating GCP CCS clusters
- Added taints to machine pool commands
- Machine pool labels and taints can be edited via `ocm edit machinepool` command
- Added interactive option to create cluster command
- Added shell completion
- Added list `ocm list orgs` command
- Updated OCM integration URL helper

## 0.1.45 Nov 22 2020

- `instance-type` is a required parameter in create machine pool command.
- Improve help and positional arg enforcement in most command.
- Show version in describe cluster command.
- Fix version check when creating a cluster.
- Add upgrade policy commands.
- Update ocm-sdk-go to 0.1.145
- Add `dry-run` parameter to create cluster command.
- Add list regions comamand.

## 0.1.44 Oct 15 2020

- Convert cluster versions to list versions
- `ocm tunnel` uses cluster id directly without a flag
- Update ocm-sdk-go to 0.1.139
- Add list/create/edit/delete machine pool commands

## 0.1.43 Sep 23 2020

- Show channel group in 'ocm describe cluster'.
- Add goreleaser config for homebrew-tap.
- Output sshuttle command execution string.
- new sub-command to show the plugins.
- Simplify cluster login via browser.
- Enable logging in via external_id.
- Add creator details.
- Support creating CCS clusters.
- Implement edit cluster command.
- Add token generation command.
- bump ocm-sdk-go to v0.1.131.

## 0.1.42 Sep 1 2020

- Display provision shard name in describe cluster
- Add more options to create cluster command
- Add `ocm tunnel` command
- Hide expiration time parameters in create cluster command
- Support git style ocm plugin

## 0.1.41 Aug 19 2020

- Assume expiration is 0 when missing 'exp' claim in the jwt token.

## 0.1.40 Aug 19 2020

- Add Product ID field to list/describe clusters.
- Add more env aliases to login command.
- Add delete identity provider command.
- Add delete ingress command.
- Add list addons command.
- Add edit ingress command.
- Usage is not displayed after error occurs.
- Bump ocm-sdk-go to 0.1.122.

## 0.1.39 Jul 9 2020

- Add support for creating a private cluster.
- Don't fail "cluster describe" if a user is unauthorized to get account.
- cluster list, create and describe are deprecated and replaced by `list clusters`,
  `create cluster` and `describe cluster`.
- Add support for creating identity providers.
- Add support for creating users.
- Add support for creating ingresses.
- Add support for listing identity providers.
- Add support for listing users.
- Add support for listing ingresses.
- Bump ocm-sdk-go to 0.1.112.

## 0.1.38 Jun 13 2020

- Add support for expiration in ocm cluster create.
- Add support for specifying cloud provider.
- Add cloud provider to default columns.
- config: beef up help message.
- Add console URL to describe.
- Output Console URL.
- Add shell completion for resources.
- Add API Listening to cluster descrribe.
- Update to ocm-sdk-go 0.1.105
- Allow setting --managed=false in cluster list.

## 0.1.37 Feb 26 2020

- Describe by name, identifier or external identifier (fixes
  [#59](https://github.com/openshift-online/ocm-cli/issues/59)).
- Support query parameters in raw HTTP methods (fixes
  [#6](https://github.com/openshift-online/ocm-cli/issues/6)).

## 0.1.36 Feb 14 2020

- Add `state` to list of default columns for cluster list.
- Preserve order of attributes in JSON output.

## 0.1.35 Feb 3 2020

- Display quota so it supports add-ons.

## 0.1.34 Jan 16 2020

- Add number of _infra_ nodes to the output of the `cluster describe` command.
- Add `--roles` flag to the `account users` command.
- Add support for `OCM_CONFIG` environment variable to indicate an alternative
  location of the configuration file.
- Tighten output of the `account orgs`, `account quota`, `account users` and
  `cluster list` commands.

## 0.1.33 Jan 8 2020

- Update to SDK 0.1.78.
- Add quota resource name.
- Tighten up list output.
- Remove redundant `href` column from organization list.
- Add parameter usage example.
- Add organization details to account status command.

## 0.1.32 Dec 12 2019

- Add shortcuts for role bindings and resource quota.
- Add shortcuts for roles and SKUs.

## 0.1.31 Dec 2 2019

- Add support for _Windows_.

## 0.1.30 Dec 2 2019

- Add `--flavour` option to `ocm cluster create`.

## 0.1.28 Nov 18 2019

- Allow bare `ocm login` to suggest the token page without extra noise.

## 0.1.28 Nov 17 2019

- Dropped support for _developers.redhat.com_.

## 0.1.27 Oct 15 2019

- Added `oc cluster versions` command.

## 0.1.26 Oct 3 2019

- Fixed the `cluster create` command so that it retrieves all the enabled
  versions.

## 0.1.25 Sep 26 2019

- Added new `cluster create` command.

- Added support for `production`, `staging` and `integration` as values of the
  `--url` parameter.

## 0.1.24 Sep 14 2019

- Fix quota output to look at correct API field.

## 0.1.23 Sep 12 2019

- Fix `login` command so that it clears old tokens.

## 0.1.22 Sep 9 2019

- Change default version field to point to current version.

- Add ability to open the console URL in browser.

## 0.1.21 Aug 28 2019

- Don't print usage message when the `get`, `post`, `patch` and `delete`
  commands receive error responses from the server.

## 0.1.20 Aug 27 2019

- Rename the tool to `ocm`.

## 0.1.19 Aug 15 2019

- Fixed issue [#62](https://github.com/openshift-online/uhc-cli/pull/62): the
  `--url` option of the `login` command should not be mandatory.

## 0.1.18 Aug 14 2019

- Improvements in the `cluster list` command, including increasing the size of
  the _name_ column.

- Added new `orgs` command to list organizations.

- Added new `account orgs` command to list organizations for the current
  account.

- Print roles of current user with the `account status` command.

- Add filter positional argument to the `cluster list` command.

## 0.1.17 Jul 2 2019

- Added the `account` command.

## 0.1.16 Jun 28 2019

- Fix deprecated issuer: should be _developers.redhat.com_ instead of
  _sso.redhat.com_.

## 0.1.15 Jun 27 2019

- Added the `--single` option to the `get` command to format the output in one
  single line.

- Improvements in the `cluster login` command.

- Changed the default authentication service from _developers.redhat.com_ to
  _sso.redhat.com_. The old service will still be used when authenticating with
  a user name and password or with token issued by _developers.redhat.com_.

## 0.1.14 Jun 20 2019

- Added the `config get` and `config set` commands to get and set configuration
  settings.

- Added support for shortcuts to the raw HTTP commands.

- Added the `whoami` command.

- Added support for custom columns in the `cluster list` command.

## 0.1.13 Jun 12 2019

- Added the `cluster login` command.

## 0.1.12 Jun 7 2019

- Improvements in the `cluster list` and `cluster describe` commands.

## 0.1.11 May 8 2019

- Added the `completion` command that generates _bash_ completion scripts.

## 0.1.10 May 3 2019

- Adapt to changes in the API and SDK that moved cluster basic metrics to a new
  `metrics` attribute.

## 0.1.9 May 2 2019

- Added the `cluster` command.

## 0.1.8 Apr 18 2019

- Update to use the new package names of the SDK and the CLI.

- Build static binary.

## 0.1.7 Apr 9 2019

- Send output to `stderr` only if the response HTTP code is greater than 400.

## 0.1.6 Mar 27 2019

- Update to SDK 0.1.3.

## 0.1.5 Mar 27 2019

- Don't pass empty tokens to connection constructor.

## 0.1.4 Mar 24 2019

- Fix printing of tokens.
- Don't reorder JSON output if `jq` is available.

## 0.1.3 Mar 24 2019

- Fix check of token expiration.

## 0.1.2 Mar 24 2019

- Add support for login with token.

## 0.1.1 Mar 14 2019

- Don't split the values of the `--parameter` command line option at commas.

## 0.1.0 Jan 24 2019

- Moved from the `api-client` project into its own `uhc-cli` project.
