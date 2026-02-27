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

	// DNS zone flags
	domainPrefixFlagDescription = `User-defined prefix for DNS zone. 
It must be unique and consist of lowercase alphanumeric characters or '-', 
start with an alphabetic character, and end with an alphanumeric character. The maximum length is 15 characters. 
Once set, the DNS zone prefix cannot be changed.`
	networkIdFlagDescription        = `ID of the Shared VPC network that will be used by the DNS zone.`
	networkProjectIdFlagDescription = `ID of the GCP project that is used as network project to VPC Network. 
This is required only for creating dns-zone`
	dnsZoneProjectIdFlagDescription = `ID of the GCP project that will be used by the DNS zone`
)
