package gcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/utils"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
)

const (
	ModeAuto   = "auto"
	ModeManual = "manual"
	// See reference: https://cloud.google.com/monitoring/api/metrics_gcp_i_o
	// iam_googleapis_com:workload_identity_federation_count and workload_identity_federation/key_usage_count
	// Both metrics could be used,
	// but the first metric was chosen because it was the default metric used by the GCP on Metrics Explorer
	WifQuery = `sum(rate(iam_googleapis_com:workload_identity_federation_count{pool_id="%s",result="success"}[%dm]))`
)

var Modes = []string{ModeAuto, ModeManual}

// Checks for WIF config name or id in input
func wifKeyArgCheck(args []string) error {
	if len(args) != 1 || args[0] == "" {
		return fmt.Errorf("expected exactly one command line parameters containing the name " +
			"or ID of the WIF config")
	}
	return nil
}

// Extracts WIF config name or id from input
func wifKeyFromArgs(args []string) (string, error) {
	if err := wifKeyArgCheck(args); err != nil {
		return "", err
	}
	return args[0], nil
}

// findWifConfig finds the WIF configuration by ID or name
func findWifConfig(client *cmv1.Client, key string) (*cmv1.WifConfig, error) {
	collection := client.GCP().WifConfigs()
	page := 1
	size := 1
	query := fmt.Sprintf(
		"id = '%s' or display_name = '%s'",
		key, key,
	)

	response, err := collection.List().Search(query).Page(page).Size(size).Send()
	if err != nil {
		return nil, err
	}
	if response.Total() == 0 {
		return nil, fmt.Errorf("WIF configuration with identifier or name '%s' not found", key)
	}
	if response.Total() > 1 {
		return nil, fmt.Errorf("there are %d WIF configurations found with identifier or name '%s'", response.Total(), key)
	}
	return response.Items().Slice()[0], nil
}

// getPathFromFlag validates the filepath
func getPathFromFlag(targetDir string) (string, error) {
	if targetDir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			return "", errors.Wrapf(err, "failed to get current directory")
		}

		return pwd, nil
	}

	fPath, err := filepath.Abs(targetDir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve full path")
	}

	sResult, err := os.Stat(fPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("directory %s does not exist", fPath)
	}
	if !sResult.IsDir() {
		return "", fmt.Errorf("file %s exists and is not a directory", fPath)
	}

	return targetDir, nil
}

// converts openshift version of form X.Y to template ID of form vX.Y
func versionToTemplateID(version string) string {
	// Check if version is a semver in the form X.Y
	re := regexp.MustCompile(`^\d+\.\d+$`)
	if re.MatchString(version) {
		return "v" + version
	}

	// Otherwise, return the version as is
	return version
}

// getFederatedProjectNumber returns the federated project number if it exists, otherwise returns the project number
func getFederatedProjectNumber(wifConfig *cmv1.WifConfig) string {
	if wifConfig.Gcp().FederatedProjectNumber() != "" && wifConfig.Gcp().FederatedProjectNumber() != "0" {
		return wifConfig.Gcp().FederatedProjectNumber()
	}
	return wifConfig.Gcp().ProjectNumber()
}

// getFederatedProjectId returns the federated project id if it exists, otherwise returns the project id
func getFederatedProjectId(wifConfig *cmv1.WifConfig) string {
	if wifConfig.Gcp().FederatedProjectId() != "" {
		return wifConfig.Gcp().FederatedProjectId()
	}
	return wifConfig.Gcp().ProjectId()
}

// updateFederatedProjectIfChanged if federated project was provided,
// and if it differs from main project, update the WIF config with the federated project.
func updateFederatedProjectIfChanged(
	ctx context.Context,
	gcpClient gcp.GcpClient,
	wifBuilder *cmv1.WifConfigBuilder,
	wifConfig *cmv1.WifConfig,
	federatedProject string,
) (bool, error) {

	if federatedProject == "" ||
		wifConfig.Gcp().FederatedProjectId() == federatedProject {
		return false, nil
	}

	projectNumInt64, err := gcpClient.ProjectNumberFromId(ctx, federatedProject)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get GCP project number from project id")
	}

	wifBuilder.Gcp(cmv1.NewWifGcp().
		FederatedProjectId(federatedProject).
		FederatedProjectNumber(strconv.FormatInt(projectNumInt64, 10)))

	return true, nil
}

func federatedProjectPoolUsageVerification(
	ctx context.Context,
	log *log.Logger,
	connection *sdk.Connection,
	wifConfig *cmv1.WifConfig,
	gcpShim GcpClientWifConfigShim,
) error {

	clustersAssociated, err := gcpShim.HasAssociatedClusters(ctx, connection, wifConfig.ID())
	if err != nil {
		return errors.Wrapf(err, "failed to check if wif-config '%s' has associated clusters. "+
			"Please try again or contact Red Hat support if the issue persists", wifConfig.ID())
	}

	// Only validate pool usage if there are associated clusters
	if !clustersAssociated {
		return nil
	}

	// Validate new pool usage
	log.Println("Ensuring new workload identity pool has recent traffic...")
	if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
		if validationErr := gcpShim.ValidateNewWifConfigPoolUsage(ctx, wifConfig); validationErr != nil {
			return true, validationErr
		}
		return false, nil
	}, IamApiRetrySeconds, log); err != nil {
		log.Printf("Timed out verifying workload identity pool usage\n" +
			"Cannot determine if old identity pool may be removed." +
			"Please consult user documentation for manually checking usage")
		return nil
	}

	// Validate old pool usage
	log.Println("Ensuring old workload identity pool is still there and not being used...")
	if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
		if err := gcpShim.ValidateOldWifConfigPoolUsage(ctx, wifConfig); err != nil {
			return true, err
		}
		return false, nil
	}, IamApiRetrySeconds, log); err != nil {
		log.Printf("Timed out verifying old workload identity pool usage\n" +
			"Cannot determine if old identity pool may be removed." +
			"Please consult user documentation for manually checking usage")
		return nil
	}

	return nil
}
