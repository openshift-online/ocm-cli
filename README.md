# OCM API Command Line Tools

This project contains the `ocm` command line tool that simplifies the use
of the _OCM_ API available at https://api.openshift.com.

## Installation

### Linux Package Manager

The preferred way to install the tool in _Fedora_ and _CentOS_ is to use the
RPM packages built in [Fedora Copr](https://copr.fedorainfracloud.org/coprs/ocm/tools).
To enable that repository and install the tool use the following commands:

```
# dnf copr enable ocm/tools
# dnf install ocm-cli
```

This will install the `ocm` command and will keep it updated using the same
mechanism used to update all the other packages of the distribution.

### MacOS Brew

```
$ brew install ocm
```

### Build From Source

If you are not using one of these distributions or you don't want to use the RPM
packages then you can alternatively get the release binaries from the _GitHub_
[releases page](https://github.com/openshift-online/ocm-cli/releases). For
example, to install version 0.1.30 to your personal `bin` directory you can use
the following commands:

```
$ mkdir -p ~/bin
$ curl -Lo ~/bin/ocm https://github.com/openshift-online/ocm-cli/releases/download/v0.1.30/ocm-linux-amd64
$ chmod +x ~/bin/ocm
```

Finally, if none of the installation options described above work for you then
you can install it using `go get` or `go install`:

```
$ go get -u github.com/openshift-online/ocm-cli/cmd/ocm
```
or
```
$ go install github.com/openshift-online/ocm-cli/cmd/ocm@latest
```

But take into account that the results of installing with `go get` depend on the
version of _Go_ that you use and on the values of certain environment variables.
It is particularly problematic to install with `go get` if the version of _Go_
used doesn't support modules, because the dependencies used may not be the ones
tested by the developers. In general installations done with `go get` aren't
supported or recommended.

## Activating shell completions

Run the following to see instructions for various shells:

```
$ ocm completion --help
```

## Log In

The first step to use the tool is to log-in with your OpenShift Cluster Manager
offline access token which you can get below:

https://console.redhat.com/openshift/token

To do that use the `login` command:

```
$ ocm login --token=eyJ...
```

This will use the provided token to request _OpenID_ access and refresh tokens
to _sso.redhat.com_. The tokens will be saved for future use to the
`~/.config/ocm/ocm.json` file.

Note: MacOS store the token at `~/Library/Application\ Support/ocm/ocm.json`

IMPORTANT: Before version 0.1.56 the configuration file used to be
`~/.ocm.json`. If that exists it will still be used. It is recommended to
remove it and login again, or move it to the new location. For example:

```
$ mkdir -p ~/.config/ocm
$ mv ~/.ocm.json ~/.config/ocm/ocm.json
```

The `login` command has options to log-in to other environments. For example,
if you have a service running in your local environment and you want to use the
tool to test it, you can log-in like this:

```
$ ocm login \
--token=eyJ... \
--url=https://localhost:8000 \
--insecure
```

NOTE: The `insecure` option disables verification of TLS certificates and host
names, do not use it in production environments.

## Multiple Concurrent Logins with OCM_CONFIG

An `~/config/ocm/ocm.json` file stores login credentials for a single API
server. Using multiple servers therefore requires having to log in and out a lot
or the ability to utilize multiple config files. The latter functionality is
provided with the `OCM_CONFIG` environment variable. If running `ocm login` was
successfull in both cases, the `ocm whoami` commands will return different
results:

```
$ OCM_CONFIG=$HOME/ocm.json.prod ocm login --url=production --token=...
(…)
$ OCM_CONFIG=$HOME/ocm.json.stg ocm login --url=staging --token=...
(…)
$ OCM_CONFIG=$HOME/ocm.json.prod ocm whoami
(…)
$ OCM_CONFIG=$HOME/ocm.json.stg ocm whoami
(…)
```

NOTE: Tokens for production and staging will differ.

## Obtaining Tokens

If you need the _OpenID_ access token to use it with some other tool, you can
use the `token` command:

```
$ ocm token
```

That will print the raw _OpenID_ access token, which you can then use to send
requests to the server with some other tool. For example, if you want to use
[curl](https://curl.haxx.se) to retrieve your list of clusters you can do the
following:

```
$ curl \
--header "Authorization: Bearer $(ocm token)" \
https://api.openshift.com/api/clusters_mgmt/v1/clusters
```

The details of the _OpenID_ access token, in JSON format, can be displayed using
the `--payload` option:

```
$ ocm token --payload
```

That will display the JSON representation of the access token, which is useful
to diagnose authentication issues.

## Revoking Tokens

If you've compromised your offline token, you can get it revoked like this:

1. Make sure you're logged into OCM with your browser.
2. Go [here](https://sso.redhat.com/auth/realms/redhat-external/account/#/applications).
3. Click REVOKE GRANT for the application _cloud-services_.

If you now follow the log in procedure new tokens will be generated.

## Log Out

To log out run the `logout` command:

```
$ ocm logout
```

That will remove the `~/.config/ocm/ocm.json` file, so next time you want to
use the tool you will need to log-in again. You can also remove that file
manually; the effect is exactly the same.

## Retrieving Objects

Once logged in you can use the `get` command to retrieve objects. For example,
to retrieve the list of clusters with a name that starts with `my` you can use
the following command:

```
$ ocm get /api/clusters_mgmt/v1/clusters --parameter search="name like 'my%'"
```

The `--parameter` option is used to specify query parameters. It is most useful
combined with the `get` command, but it can be also used with any other command.
For detailed information about the query parameters supported by each resource
see the [reference documentation](https://api.openshift.com).

The `search` query parameter is specially useful to retrieve objects from
collections that support searching. The syntax of this parameter is similar to
the syntax of the `where` clause of an SQL statement, but using the names of the
attributes of the object instead of the names of the columns of a table. For
example, in order to retrieve the clusters with a name starting with `my` and
created in a DNS domain ending with `example.com` the complete command can be
the following:

```
$ ocm get /api/clusters_mgmt/v1/clusters \
--parameter search="name like 'my%' and dns.base_domain like '%.example.com'"
```

To find the AWS regions in the US:

```
$ ocm get /api/clusters_mgmt/v1/cloud_providers/aws/regions \
--parameter search="display_name like 'US %'"
```

To find the clusters created after March 1st 2019:

```
$ ocm get /api/clusters_mgmt/v1/clusters \
--parameter search="creation_timestamp >= '2019-03-01'"
```

To find the clusters that are either ready or installing:

```
$ ocm get /api/clusters_mgmt/v1/clusters \
--parameter search="state in ('ready', 'installing')"
```

The result of that will be a JSON document containing the description of those
clusters, for example:

```json
{
  "kind": "ClusterList",
  "page": 1,
  "size": 6,
  "total": 10
  "items": [
    {
      "kind": "Cluster",
      "id": "1GUAUWE3E1IS87Q99M0kxO1LpCG",
      "href": "/api/clusters_mgmt/v1/clusters/1GUAUWE3E1IS87Q99M0kxO1LpCG",
      "name": "mycluster",
      "api": {
        "url": "https://mycluster-api.example.com:6443"
      },
      "console": {
        "url": "https://console-openshift-console.apps.mycluster.example.com"
      },
      ...
    },
    ...
  ]
}
```

As the server will always return JSON documents it is very convenient to use the
[jq](https://stedolan.github.io/jq) tool to extract the information that you
need. For example, if you want to get the list of identifiers of your clusters
you can do the following:

```
$ ocm get /api/clusters_mgmt/v1/clusters | jq -r .items[].id
```

That will return something like this:

```
1FtmglZGw2byDzO8tb2cCtWxCNf
1FtRj13Fz2DIcm4zaDrcLvKAIyf
...
```

The `get` command can also be used to retrieve information from sub-resources
associated to objects. For example, the credentials of a cluster (SSH keys,
administrator password and _kubeconfig_) are available in a `credentials`
sub-resource. So if your cluster identifier is `123` you can retrieve the
credentials with this command:

```
$ ocm get /api/clusters_mgmt/v1/clusters/123/credentials
```

Again the [jq](https://stedolan.github.io/jq) tool is very useful here. For
example, it can be used to extract the _kubeconfig_ to a file that can then be
used directly with the `oc` command:

```
$ # Get the file:
$ ocm get /api/clusters_mgmt/v1/clusters/123/credentials \
| jq -r .kubeconfig > mycluster.config

$ # Use it:
$ oc --config=mycluster.config get pods
```

For a complete definition of the types of objects, and their attributes, see the
[reference documentation](https://api.openshift.com).

## Creating Objects

To create objects use the `post` command, and put the JSON representation of the
object either in the standard input or else in a file indicated by the `--body`
option. For example, to create a new managed cluster prepare a `mycluster.json`
file with this content:

```json
{
  "name": "mycluster",
  "flavour": {
    "id": "osd-4"
  },
  "region": {
    "id": "us-east-1"
  },
  "managed": true
}
```

And then use the `post` command:

```
$ ocm post /api/clusters_mgmt/v1/clusters < mycluster.json
```

Or with the `--body` option:

```
$ ocm post /api/clusters_mgmt/v1/clusters --body=mycluster.json
```

That will send the request to the server, which will initiate the process of
creating the object, and will return a JSON document containing the
representation.

Complicated objects, like a cluster, are usually created asynchronously, so the
fact that the server returns a response doesn't mean that the object is ready to
use. Clusters, for example, have a `state` attribute to indicate that. So after
creating a cluster you will have to periodically check till the cluster is
ready. To do so first get the `id` returned by the `post` command:

```
$ ocm post /api/clusters_mgmt/v1/clusters --body=mycluster.json | jq -r .id
```

Then use that identifier to check the value of the `state` attribute, till it
is `ready`:

```
$ ocm get /api/clusters_mgmt/v1/clusters/123 | jq -r .state
```

## Deleting Objects

Objects can be deleted using the `delete` command. For example to delete the
cluster with identifier `123` use the following command:

```
$ ocm delete /api/clusters_mgmt/v1/clusters/123
```

Some objects can be deleted in different ways. For example, a cluster can be
deleted completely, destroying all the virtual machines, disks and any other
resources it uses. But it can also just be deleted from the database while
preserving the virtual machines, disks, etc. To do so the server accepts a
`deprovision` parameter, which can be `true` or `false`. To use it with the tool
add the `--parameter` option. For example, to delete the cluster with identifier
`123` only from the database, use the following command:

```
$ ocm delete /api/clusters_mgmt/v1/clusters/123 --parameter "deprovision=false"
```

Deletion, like creation, is a lengthy process for complicated objects like
clusters, and it happens asynchronously. After the `delete` command finishes it
will take some time to actually delete the cluster. That can be checking using
the `get` command till it returns a `404 Not Found` response.

## Config

The configuration variables can be read and set via the `get` and `set`
commands. These settings will be persisted in the `~/.config/ocm/ocm.json`
file in your home directory.

```
$ ocm config get url
```

```
$ ocm config set url https://api.openshift.com
```

## Building RPMs

Currently RPMs are built for _Fedora_ and _CentOS_ using
[Fedora Copr](https://copr.fedorainfracloud.org/coprs/ocm/tools).

The mechanism selected to do the build is a the following custom script that
generates the RPM `.spec` file:

```
# Check that the event payload exists:
if [[ ! -f hook_payload ]]; then
    echo "Event payload file 'hook_payload' doesn't exist"
    exit 1
fi

# Check that the event is the creation of a tag:
ref_type=$(cat hook_payload | jq -r .ref_type)
if [[ "${ref_type}" != "tag" ]]; then
    echo "Expected reference type 'tag' but got '${ref_type}'"
    exit 1
fi

# Check that the tag is well formed:
ref=$(cat hook_payload | jq -r .ref)
if [[ ! "${ref}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Reference '${ref}' isn't well formed"
    exit 1
fi

# Set the version to use:
version="${ref:1}"

# Generate the .spec file:
cat > ocm-cli.spec.in <<"."
%global debug_package %{nil}

Name: ocm-cli
Version: @version@
Release: 1%{?dist}
Summary: CLI for the Red Hat OpenShift Cluster Manager
License: ASL 2.0
URL: https://github.com/openshift-online/ocm-cli
Source: https://github.com/openshift-online/ocm-cli/archive/v@version@.tar.gz

# We need to download Go explicitly because in most of the platforms that we
# use the version available is too old.
%define go_tar https://golang.org/dl/go1.16.8.linux-amd64.tar.gz
%define go_sum f32501aeb8b7b723bc7215f6c373abb6981bbc7e1c7b44e9f07317e1a300dce2

BuildRequires: curl
BuildRequires: git
BuildRequires: make

%description
CLI for the Red Hat OpenShift Cluster Manager

%prep
%setup

%build

# Create the Go directories:
export GOROOT="${PWD}/.goroot"
export GOPATH="${PWD}/.gopath"
mkdir "${GOROOT}" "${GOPATH}"
PATH="${GOROOT}/bin:${PATH}"

# Download and install Go:
curl --location --output go.tar.gz %{go_tar}
echo %{go_sum} go.tar.gz | sha256sum --check
tar --directory "${GOROOT}" --extract --strip-components 1 --file go.tar.gz

# Build the binary:
make

%install
install -m 0755 -d %{buildroot}%{_bindir}
install -m 0755 ocm %{buildroot}%{_bindir}

%files
%license LICENSE.txt
%doc README.md
%{_bindir}/*
.
sed \
  -e "s/@version@/${version}/g" \
  < ocm-cli.spec.in \
  > ocm-cli.spec

# Bye:
exit 0
```

If this script needs to be changed you will need to go to the _copr_ user
interface and update it manually.

The _GitHub_ repository is configured with a webhook that will trigger the
_copr_ build when a new tag is pushed to the repository.

The _build dependencies_ section of the _copr_ configuration should include the
`jq` package is it is needed to extract the version number from the payload of
the event sent by the _GitHub_ webhook.

## Extend ocm with plugins

Just like how
[kubectl plugins](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins)
works, you can write your own ocm plugins and put the binary under the
$PATH directory, the plugin name should be named with prefix `ocm-`, like
`ocm-foo`.
