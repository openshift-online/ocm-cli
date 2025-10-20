package gcp

const (
	modeFlagDescription = `How to perform the operation. Valid options are:
auto (default): Resource changes will be automatically applied using the
                current GCP account.
manual:         Commands necessary to modify GCP resources will be output
                as a script to be run manually.
`

	targetDirFlagDescription        = `Directory to place generated files (defaults to current directory)`
	versionFlagDescription          = `Version of OpenShift to configure the WIF resources for`
	federatedProjectFlagDescription = `ID of the Google cloud project that will host the WIF pool`
)
