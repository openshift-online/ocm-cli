package gcp

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func TestCreateScript(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	Expect(err).To(BeNil())

}

// 	// Create a temporary directory for testing
// 	tempDir, err := ioutil.TempDir("", "test")
// 	assert.NoError(t, err)
// 	defer os.RemoveAll(tempDir)

// 	// Create a mock WifConfigOutput
// 	wifConfig := &models.WifConfigOutput{
// 		Status: &models.WifConfigStatus{
// 			WorkloadIdentityPoolData: &models.WorkloadIdentityPoolData{
// 				Jwks: "mock-jwks",
// 			},
// 		},
// 	}

// 	// Call the createScript function
// 	err = createScript(tempDir, wifConfig)
// 	assert.NoError(t, err)

// 	// Verify that the script.sh file was created
// 	scriptPath := filepath.Join(tempDir, "script.sh")
// 	_, err = os.Stat(scriptPath)
// 	assert.NoError(t, err)

// 	// Verify the content of the script.sh file
// 	scriptContent, err := ioutil.ReadFile(scriptPath)
// 	assert.NoError(t, err)
// 	expectedScriptContent := "#!/bin/bash\n" +
// 		"# Create a workload identity pool\n" +
// 		"...\n" +
// 		"# Create a workload identity provider\n" +
// 		"...\n" +
// 		"# Create service accounts:\n" +
// 		"...\n"
// 	assert.Equal(t, expectedScriptContent, string(scriptContent))

// 	// Verify that the jwk.json file was created
// 	jwkPath := filepath.Join(tempDir, "jwk.json")
// 	_, err = os.Stat(jwkPath)
// 	assert.NoError(t, err)

// 	// Verify the content of the jwk.json file
// 	jwkContent, err := ioutil.ReadFile(jwkPath)
// 	assert.NoError(t, err)
// 	expectedJwkContent := "mock-jwks"
// 	assert.Equal(t, expectedJwkContent, string(jwkContent))
// }
