package ocm

import (
	"bytes"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/openshift-online/ocm-common/pkg/ocm/consts"
)

// MockResponse implements the ResponseWithHeaders interface for testing
type MockResponse struct {
	headers http.Header
}

func (m *MockResponse) Header() http.Header {
	return m.headers
}

func TestHandleDeprecationWarning(t *testing.T) {
	// Capture stderr for testing
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Test with OCM deprecation message header (master message)
	headers := http.Header{}
	headers.Set(consts.DeprecationHeader, "1234567890")
	headers.Set(consts.OCMDeprecationMessage, "This endpoint is deprecated. Use /v2/ instead.")

	response := &MockResponse{
		headers: headers,
	}

	HandleDeprecationWarningFromTypedResponse(response)

	// Restore stderr
	w.Close()
	os.Stderr = old

	// Read the output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	expected := "Warning: This endpoint is deprecated. Use /v2/ instead.\n"
	if output != expected {
		t.Errorf("Expected: %q, Got: %q", expected, output)
	}
}

func TestHandleDeprecationWarningGeneric(t *testing.T) {
	// Capture stderr for testing
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Test with only deprecation header (no custom message)
	headers := http.Header{}
	headers.Set(consts.DeprecationHeader, "true")

	response := &MockResponse{
		headers: headers,
	}

	HandleDeprecationWarningFromTypedResponse(response)

	// Restore stderr
	w.Close()
	os.Stderr = old

	// Read the output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	expected := "Warning: Deprecated endpoint was used\n"
	if output != expected {
		t.Errorf("Expected: %q, Got: %q", expected, output)
	}
}

func TestHandleDeprecationWarningFutureTimestamp(t *testing.T) {
	// Capture stderr for testing
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Test with future timestamp (use UTC)
	futureTime := time.Now().UTC().Add(24 * time.Hour)
	headers := http.Header{}
	headers.Set(consts.DeprecationHeader, futureTime.Format(time.RFC3339))

	response := &MockResponse{
		headers: headers,
	}

	HandleDeprecationWarningFromTypedResponse(response)

	// Restore stderr
	w.Close()
	os.Stderr = old

	// Read the output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !bytes.Contains(buf[:n], []byte("Warning: This endpoint will be deprecated on")) {
		t.Errorf("Expected future deprecation warning, Got: %q", output)
	}
}

func TestHandleDeprecationWarningNoHeader(t *testing.T) {
	// Capture stderr for testing
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Test with no deprecation header
	response := &MockResponse{
		headers: http.Header{},
	}

	HandleDeprecationWarningFromTypedResponse(response)

	// Restore stderr
	w.Close()
	os.Stderr = old

	// Read the output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Should be empty since no deprecation header was present
	if output != "" {
		t.Errorf("Expected no output, Got: %q", output)
	}
}

func TestHandleDeprecationWarningSDKResponse(t *testing.T) {
	// Test that the function works with nil response
	HandleDeprecationWarning(nil)
	// This should not panic and should not output anything
}
