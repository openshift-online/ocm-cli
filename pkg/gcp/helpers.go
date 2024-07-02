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

func (c *gcpClient) fmtSaResourceId(accountId, projectId string) string {
	return fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectId, accountId, projectId)
}
