package gcp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
)

const (
	ModeAuto   = "auto"
	ModeManual = "manual"
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
