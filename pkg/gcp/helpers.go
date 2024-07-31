package gcp

import (
	"fmt"
)

func (c *gcpClient) fmtSaResourceId(accountId, projectId string) string {
	return fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectId, accountId, projectId)
}
