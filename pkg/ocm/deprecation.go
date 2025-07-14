package ocm

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openshift-online/ocm-common/pkg/ocm/consts"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

// ResponseWithHeaders represents any response type that has headers
type ResponseWithHeaders interface {
	Header() http.Header
}

// RequestInterface represents any request that can be sent
type RequestInterface interface {
	Send() (*sdk.Response, error)
}

// TypedRequestInterface represents typed requests (like Get, Post, etc.)
type TypedRequestInterface[T any] interface {
	Send() (T, error)
}

// SendAndHandleDeprecation wraps a request.Send() call and automatically handles deprecation warnings
func SendAndHandleDeprecation(request RequestInterface) (*sdk.Response, error) {
	response, err := request.Send()
	if err != nil {
		return response, err
	}

	HandleDeprecationWarning(response)
	return response, nil
}

// SendTypedAndHandleDeprecation wraps typed request.Send() calls and automatically handles deprecation warnings
func SendTypedAndHandleDeprecation[T ResponseWithHeaders](request TypedRequestInterface[T]) (T, error) {
	response, err := request.Send()
	if err != nil {
		return response, err
	}

	HandleDeprecationWarningFromTypedResponse(response)
	return response, nil
}

// HandleDeprecationWarning checks for deprecation headers in the HTTP response
// and prints appropriate warning messages.
func HandleDeprecationWarning(response *sdk.Response) {
	if response == nil {
		return
	}
	HandleDeprecationWarningFromHeaders(response.Header)
}

// HandleDeprecationWarningFromHeaders checks for deprecation headers and prints warnings
func HandleDeprecationWarningFromHeaders(getHeader func(string) string) {
	if getHeader == nil {
		return
	}

	// Check for the deprecation header
	deprecationHeader := getHeader(consts.DeprecationHeader)
	if deprecationHeader == "" {
		// No deprecation header found
		return
	}

	// Check for the OCM deprecation message header first (master message)
	deprecationMessage := getHeader(consts.OCMDeprecationMessage)
	if deprecationMessage != "" {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", deprecationMessage)
		return
	}

	// Parse the deprecation header to check if it's a future timestamp
	deprecationHeader = strings.TrimSpace(deprecationHeader)
	if deprecationHeader == "" {
		fmt.Fprintf(os.Stderr, "Warning: Deprecated endpoint was used\n")
		return
	}

	// Try to parse as RFC3339 timestamp
	if timestamp, err := time.Parse(time.RFC3339, deprecationHeader); err == nil {
		now := time.Now().UTC()
		if timestamp.After(now) {
			// Future deprecation
			fmt.Fprintf(os.Stderr, "Warning: This endpoint will be deprecated on %s\n", timestamp.Format(time.RFC3339))
		} else {
			// Past deprecation
			fmt.Fprintf(os.Stderr, "Warning: Deprecated endpoint was used\n")
		}
		return
	}

	// If we can't parse the timestamp, show generic message
	fmt.Fprintf(os.Stderr, "Warning: Deprecated endpoint was used\n")
}

// HandleDeprecationWarningFromTypedResponse handles deprecation warnings from typed responses
func HandleDeprecationWarningFromTypedResponse(response ResponseWithHeaders) {
	if response == nil {
		return
	}

	headers := response.Header()
	getHeader := func(name string) string {
		return headers.Get(name)
	}

	HandleDeprecationWarningFromHeaders(getHeader)
}
