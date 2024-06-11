package gcp

import (
	"fmt"
	"strings"
)

func (c *gcpClient) extractEmail(saResourceId string) string {
	email := strings.SplitAfter(saResourceId, "/serviceAccounts/")
	if len(email) != 2 {
		return ""
	}
	return email[1]
}

func (c *gcpClient) extractProject(saResourceId string) string {
	projectLevel := strings.SplitAfter(saResourceId, "projects/")
	if len(projectLevel) != 2 {
		return ""
	}
	project := strings.Split(projectLevel[1], "/")
	if len(project) < 1 {
		return ""
	}
	return project[0]
}

func (c *gcpClient) extractSecretProject(secretResource string) string {
	resources := strings.Split(secretResource, "/secrets/")
	if len(resources) != 2 {
		return ""
	}
	return resources[0]
}

func (c *gcpClient) extractSecretName(secretResource string) string {
	resources := strings.Split(secretResource, "/secrets/")
	if len(resources) != 2 {
		return ""
	}
	return resources[1]
}

func (c *gcpClient) fmtSaResourceId(accountId, projectId string) string {
	return fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectId, accountId, projectId)
}
