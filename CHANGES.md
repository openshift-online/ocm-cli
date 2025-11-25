# Changes

This document describes the relevant changes between releases of the
`ocm` command line tool.

## 1.0.9 Nov 25 2025

- Update README.md
- Build a source container (#874)
- update wif config to enable support for federated project (#884)
- pool usage after update (#888)
- Revert "pool usage after update (#888)" (#901)
- Red Hat Konflux update ocm-cli (#882)
- update golang version (#899)
- chore(deps): update konflux references
- added OWNERS file (#937)
- OCM-2093 | fix: misaligned list users result
- OCM-2093 | fix: check eventual error raised by tabwriter's flush operation

## 1.0.8 Sep 11 2025

- feat: Add CLAUDE.md configuration file
- support resource-scoped permissions wif-config (#860)

## 1.0.7 Aug 6 2025

-8ba748b wif config name on cluster review and describe (#816)
-be14307 Add deprecation header handling
-dd6c671 Handle deprecation header in all CLI commands
-f10fefa Add Makefile target to install ginkgo and skip test in case 'pass' is not available
-8593695 Remove redundant documentation msg
-ccfbfe1 Fix linter issues
-c12b3f4 availability zone help text flag to multiple cloud providers (#815)
-53547a9 Handle deprecation header in all missed CLI commands
-90b8edc Update Konflux references
-d51c67b Revert "Handle deprecation header in all missed CLI commands"
-27e48a0 Revert "Handle deprecation header in all CLI commands"
-784b40e Fix revert conflicts
-61e3381 OCM-17277 | feat: Use deprecation transport to handle backend API deprecation
-443db81 OCM-17277 | feat: bump ocm-common and run go mod tidy
-676d290 update sdk to latest
-e8580f9 cross project wif config support (#831)
-069c3bf cross projects wif-config update for second milestone (#840)
-499d78f update tekton references

## 1.0.6 Jun 4 2025

-f95acd1 Update github.com/golang/groupcache digest to 2c02b82
-a4c0ee2 Update github.com/jackc/pgservicefile digest to 5a60cdf
-2f87995 Update Konflux references (#734)
-4a1708e updates to konflux pipeline for 1.0.5 (#756)
-687527d Bump github.com/openshift-online/ocm-sdk-go from 0.1.463 to 0.1.465
-de7adc6 remove marketplace-rhm option from subscription-type options (#773)
-436ff34 secure-boot-for-shielded-vms flag for create machinepool (#778)
-3440bb5 OCM-15127 | Add make binary in ocm-cli image (#779)
-64ca7ac secure-boot-for-shielded-vms flag tests (#780)
-822e0f2 Bump github.com/MicahParks/jwkset from 0.5.20 to 0.7.0 (#728)
-8474de0 Update Konflux references
-2ecdf68 Bump github.com/spf13/cobra from 1.7.0 to 1.9.1 (#748)
-ea64448 Bump github.com/golang/glog from 1.2.4 to 1.2.5

## 1.0.5 Apr 4 2025

-8e28c06 OCM-14358 | unbind service accounts when deleting wif-configs (#723)
-62f7db3 Improving UX around gcloud iam api consistency model (#725)
-a8ab357 OCM-14357 | Ensuring wif commands are resilient to GCP's consistency model (#726)
-c41ad70 Update module github.com/golang-jwt/jwt/v4 to v4.5.2 (#727)
-19bb541 Update module github.com/hashicorp/go-version to v1.7.0 (#724)
-d4b073f Update Konflux references (#712)
-3d30c1c Update module github.com/spf13/pflag to v1.0.6 (#717)
-5010823 Update module github.com/openshift-online/ocm-sdk-go to v0.1.463 (#715)
-22467d1 various cve updates (#731)
-7deabe5 display versions on wif-config describe (#732)

## 1.0.4 Feb 27 2025

-3a8ef75 OCM-12971 | 'ocm gcp update wif-config' remediates all wif-config misconfigurations (#696)
-d4deb29 Add 'version' flag to wif-config create and update commands (#698)
-f079176 updated help message for wif verification errors
-1fb1d56 Refactored GCP client operations to log user messages and optimized resource updation (#700)
-dd04c33 Update Konflux references (#697)
-c84225e OCM-11995 | feat : Add GCP KMS custom encryption support (#701)
-7ac7051 added 'availability-zone' argument to machine pool creation (#703)
-56ca538 updates to konflux_build_pipeline (#704)
-b37893c listing wif-configs shows supported versions (#706)
-7087572 Update Konflux references (#702)
-7aac46f Update Konflux references (#707)
-2045f98 Update github.com/pkg/browser digest to 5ac0b6a (#708)
-0b6615a Update module github.com/golang-jwt/jwt/v4 to v4.5.1 (#709)
-1cf3d22 Update module github.com/openshift-online/ocm-sdk-go to v0.1.459 (#711)
-579f8de Update module github.com/golang/glog to v1.2.4 (#710)
-c5c95d5 n-3 vesion supportfor  wif-update (#713)

## 1.0.3 Dec 9 2024

-9fbb753 Update to use addon service API for addons function
-1bafd65 Add command 'gcp verify wif-config' (#691)
-5d0bec1 Update Konflux references (#682)
-f0ed911 OCM-12467 | feat : Updates for binaries-release-pipeline (#693)
-c42c641 Update Konflux references (#694)

## 1.0.2 Oct 25 2024

-8b70707 Release v0.1.76 (#674)
-2ef1e1d Update Konflux references to 67f0290 (#676)
-c63247f Show all WIF configs in interactive dropdown (#677)
-f1d29bc adding interactive move for WifConfig creation (#675)
-1382a6c Require provider, do not default to AWS, and check for provider-specific flags (#678)
-d0b58b4 Do not default GCP authentication type (#679)
-db35759 improve psc cli UX (#681)
-862b072 OCM-11993 | Describe cluster shows WifConfig data (#683)
-4053505 Add PSC-XPN to cluster  description (#685)
-0c456f8 OCM-10728 | interface improvements (#686)
-1e27ded Add more descriptions to WIF resources (#684)
-cdf6466 update filenames in konflux release  container (#687)
-010573f remove version from shasum (#688)

## 0.1.76 Oct 15 2024

-e034b6b Update Konflux references to 2418e94
-5066ea0 Filter wif configs in interactive mode (#660)
-878f5e3 Initial refactor to prepare to move the connection builder and config packages to ocm-common
-1ea2e05 lint
-2c66dc0 removes redundant api url
-65bf8cf Add role prefix flag on create wif-config (#662)
-a39ce2e Grant access to support group during WifConfig creation (#663)
-0275d67 Revert "Grant access to support group during WifConfig creation (#663)" (#664)
-7cddc94 Wif creation improvements, including logic to grant support access as part of wif creation. (#666)
-7f41626 Update Konflux references
-b9a750c UpdatesToKonflux (#668)
-e4aa770 OCM-10615 | Implement 'gcp wif-config update' command (#667)
-cf6e500 Dry-run wif config delete before tearing down cloud resources (#670)
-e18ea10 OCM-11842 | feat: Updates to support GCP-PSC clusters (#672)
-893acd5 wif-enable gcp-inquiries (#673)
-664b2c4 Replace wif dry-run flag with mode (#671)
-df87894 Update Konflux references (#669)

## 0.1.75 Aug 8 2024

-416843e OSD-24332 Adding CNI Type to the printed output.
-ca71863 Introduce gcp WIF sub-commands to manage wif-configs (#619)
-5f9697b multi arch release images (#631)
-951d7cd Red Hat Konflux update ocm-cli (#633)
-2604647 Limit Konflux Pipeline Runs (#634)
-9645301 Update Konflux references (#635)
-c797dfb Update Konflux references to 0dc3087
-28b521d support hermetic build (#636)
-3117d6b Update Konflux references to 9eee3cf
-d228140 Update Konflux references to 71270c3
-0ff233b update konflux pipeline (#641)
-ae2093b Update Konflux references
-2ae4aa0 Update Konflux references
-bdd172b Update Konflux references to f93024e
-d750acc Red Hat Konflux update ocm-cli Signed-off-by: red-hat-konflux <konflux@no-reply.konflux-ci.dev>
-0bbcf6e Update Konflux references
-21ff6b8 Replace wif models and client with sdk (#643)
-c3d52e2 Update Konflux build (#651)
-8073ef8 release_version (#652)
-e9a014d Update Konflux references
-78317e9 Add 'wif-config' flag as cluster create option
-49f4e41 Set project number on wif config creation
-e441c1b Update Konflux references
-ca8d9db Support listing and parameters in 'gcp get wif-config' (#656)

## 0.1.74 Jul 2 2024

- 3423d52 OCM-1888: Add docs for ocm delete account subcommands and arguments
- 42a6c63 OCM-4965: Keyring configuration storage (#600)
- 01c0241 OCM-6528 | feat: add describe ingress cmd
- debb035 OCM-8013| feat: Dockerfile for Konflux builds
- 79d7322 konflux-tekton build pipelines
- e79a4bf Red Hat Konflux update ocm-cli
- 5ea7344 Trigger pipeline only if event title starts with Konflux
- edd560c Bump ocm-sdk-go to v0.1.422 (#620)
- 5e4c99b Adds ability to change api url via env var (#621)
- 2a1f92e OCM-1398: support 'user' as variable for ocm config
- a248a57 Update Konflux references (#617)
- 8448029 Update Konflux references to fa168cd (#623)
- 441189f Red Hat Konflux purge ocm-cli (#624)

## 0.1.73 Apr 2 2024

- 165b90e OCM-4783 | feat: display warnings after cluster creation
- 0973f7b Display a cluster history URL in cluster describe
- ad98440 Fix minor formatting issue with cluster describe
- ea1c988 OCM-4962 | Feat | Add OAuth login using PKCE (#590)
- 98944f7 OCM-5759 | feat: Add Device Code Flow (#591)
- ff1c142 Display only platform-relevant field in cluster describe
- fab7ccf OCM-5281 | Feat | Add region validation from ocm-shards and list regions command (#586)
- f279dc7 Use Hostname() to set --hosted-domain
- d0f8459 improving ocm login and ocm list rh-region url resolution to reuse the url saved in config before falling back to api.openshift.com
- d894c2a OCM-6407 | edit option sends an empty payload
- 9cf11ef OCM-6450 | No update cluster with empty config
- 1caf2d0 OCM-5941 | add enable delete protection parameter
- a056c70 OCM-6140 | feat: allow customization of the domain prefix when creating a cluster
- 5aa159f OCM-6030 | chore: bump sdk to v0.1.407
- 03500fe OCM-6140: make name width to be 54 chars to fix truncation issue in case of longer name > 28 chars
- 6d1fd05 OCM-6030 | feat: allow to edit component routes of ingress
- 383d362 Make auth and device code flags visible

## 0.1.72 Dec 11 2023

- OCM-4960: Do not print expiration timestamp if not set
- OCM-5131 | feat: add SG IDs to describe cluster and list machinepools
- Add 'secure-boot-for-shielded-vms' flag to expose secureboot on GCP
- OCM-5131 | fix: don't show additional compute SGs describing cluster

## 0.1.71 Nov 15 2023

- OCM-4750 | feat: additional security group ids attributes

## 0.1.70 Oct 13 2023

- 4dad47f OCM-3510 | fix: allow clusters to edit ingresses
- 57855bf fixed setting isGcpMarketplaceSubscriptionType for non interactive mode
- 557a66b making GCP service account file mandatory for CCS clusters (#565)
- 1f5481b Add GCP marketplace terms and conditions for marketplace GCP clusters
- 47cd35c showing error and re-prompting service-account-file question if one is not provided (#567)
- 0d3f4c3 OCM-4186 | Feat | Changed marketplace-gcp-terms error message for non-interactive mode (#569)
- 455f27e OCM-4184 | Feat | Convert relative path containing tilde for service account file (#568)

## 0.1.69 Sep 29 2023

- 447488d Added default values for CIDRs and host prefix
- 1f2b39b fix duplicate machine pool information printed for the same cluster
- f0fadd8 add tests for list machinepools command
- 5b60bbf added failure tests for list machinepools command
- 498c54e Filtered OCM versions for marketplace gcp clusters

## 0.1.68 Sep 12 2023

- Bump k8s.io/apimachinery from 0.27.2 to 0.27.3
- Bump github.com/onsi/ginkgo/v2 from 2.9.7 to 2.11.0
- Bump golang.org/x/text from 0.9.0 to 0.11.0
- Bump golang.org/x/term from 0.8.0 to 0.10.0
- Bump github.com/AlecAivazis/survey/v2 from 2.3.6 to 2.3.7
- Bump github.com/openshift-online/ocm-sdk-go from 0.1.344 to 0.1.367
- Improve GetCluster message when Subscription exists but is inactive
- OCM-2657 | feat: day1/2 operations for managed ingress attributes
- Bump github.com/openshift/rosa from 1.2.22 to 1.2.24
- OCM-2941| fix: Adjusting help usage and ingress builder call
- OCM-2942 | fix: adjust help usage for cluster routes attributes
- OCM-2966 : Feat : Added subscription_type parameter to create cluster command
- OCM-3061 | fix: allow to reset route selectors/excluded namespaces

## 0.1.67 Jun 14 2023

- Bump github.com/onsi/ginkgo/v2 from 2.8.1 to 2.9.1
- Add display name to describe output
- Bump github.com/onsi/gomega from 1.27.3 to 1.27.6
- Update linting configuration
- Add HCP status to list/describe output
- Bump k8s.io/apimachinery from 0.26.1 to 0.27.1
- Bump github.com/openshift/rosa from 1.2.15 to 1.2.17
- Bump golang.org/x/term from 0.6.0 to 0.7.0
- Bump github.com/spf13/cobra from 1.6.1 to 1.7.0
- Update _copr_ build instructions
- Bump github.com/onsi/ginkgo/v2 from 2.9.2 to 2.9.7
- Bump k8s.io/apimachinery from 0.27.1 to 0.27.2
- Bump github.com/openshift-online/ocm-sdk-go from 0.1.330 to 0.1.344
- Bump github.com/openshift/rosa from 1.2.17 to 1.2.22
- OCM-2177 | Add additional status details to the describe cluster output

## 0.1.66 Feb 28 2023

- docs: update the installation w/ 'go install'
- Update brew installation instructions
- Add dependabot config
- Update github actions
- Upgrade dependencies
- Bump github.com/spf13/cobra from 1.5.0 to 1.6.1
- Bump github.com/openshift-online/ocm-sdk-go from 0.1.306 to 0.1.308
- Update revocation link
- Remove validation for GCP+private clusters
- Add build artifact for darwin/arm64
- Bump k8s.io/apimachinery from 0.24.3 to 0.26.1
- Bump github.com/onsi/gomega from 1.24.2 to 1.25.0
- Bump github.com/onsi/ginkgo/v2 from 2.7.0 to 2.8.0
- Bump golang.org/x/text from 0.6.0 to 0.7.0
- Bump golang.org/x/term from 0.4.0 to 0.5.0
- Bump github.com/onsi/gomega from 1.25.0 to 1.26.0
- Bump github.com/openshift-online/ocm-sdk-go from 0.1.308 to 0.1.316
- Bump github.com/openshift/rosa from 1.2.11 to 1.2.15
- Bump golang.org/x/net from 0.5.0 to 0.7.0
- Bump github.com/golang-jwt/jwt/v4 from 4.4.3 to 4.5.0
- Return the service cluster associated with a hypershift cluster in ocm describe cluster
- Bump github.com/onsi/gomega from 1.26.0 to 1.27.1

## 0.1.65 Dec 16 2022

- added GetLimitedSupportReasons function to allow cluster objects to access them easier
- Removed DisplayName/display_name from cluster
- network: Ensure there is no default network type
- Added name = '%s'
- Bump golang dependencies
- Upgrade linter version
- Fix linting errors
- Add no-proxy attribute to OCM-CLI
- adding an error when proxy is set for non byo-vpc cluster creation
- Add validationn when user creates a cluster onlt with no-proxy (non-interacitve mode)
- Swap flags to match usage
- Bump golang dependencies
- Fix GetCluster AMS search
- Fix edit cluster command
- Bump dependencies on ocm-sdk-go and rosa
- Add field for Management Cluster in describe cluster for hypershift clusters

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
