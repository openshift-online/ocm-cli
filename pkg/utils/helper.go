package utils

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"time"
)

// the following regex defines four different patterns:
// first pattern is to validate IPv4 address
// second,is for IPv4 CIDR range validation
// third pattern is to validate domains
// and the fourth petterrn is to be able to remove the existing no-proxy value by typing empty string ("").
// nolint
var UserNoProxyRE = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/(3[0-2]|[1-2][0-9]|[0-9]))$|^(.?[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$|^""$`)

func ValidateHTTPProxy(val interface{}) error {
	if httpProxy, ok := val.(string); ok {
		if httpProxy == "" {
			return nil
		}
		url, err := url.ParseRequestURI(httpProxy)
		if err != nil {
			return fmt.Errorf("Invalid 'proxy.http_proxy' attribute '%s'", httpProxy)
		}
		if url.Scheme != "http" {
			return fmt.Errorf("%s", "Expected http-proxy to have an http:// scheme")
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

// IsURL validates whether the given value is a valid URL
func IsURL(val interface{}) error {
	if val == nil {
		return nil
	}
	if s, ok := val.(string); ok {
		if s == "" {
			return nil
		}
		_, err := url.ParseRequestURI(fmt.Sprintf("%v", val))
		return err
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

func ValidateAdditionalTrustBundle(val interface{}) error {
	if additionalTrustBundleFile, ok := val.(string); ok {
		if additionalTrustBundleFile == "" {
			return nil
		}
		cert, err := os.ReadFile(additionalTrustBundleFile)
		if err != nil {
			return err
		}
		additionalTrustBundle := string(cert)
		if additionalTrustBundle == "" {
			return fmt.Errorf("%s", "Additional trust bundle file is empty")
		}
		additionalTrustBundleBytes := []byte(additionalTrustBundle)
		if !x509.NewCertPool().AppendCertsFromPEM(additionalTrustBundleBytes) {
			return fmt.Errorf("%s", "Failed to parse additional trust bundle")
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

func MatchNoPorxyRE(noProxyValues []string) error {
	for _, v := range noProxyValues {
		if !UserNoProxyRE.MatchString(v) {
			return fmt.Errorf("expected a valid user no-proxy value: '%s' matching %s", v,
				UserNoProxyRE.String())
		}
	}
	return nil
}

func HasDuplicates(valSlice []string) (string, bool) {
	visited := make(map[string]bool)
	for _, v := range valSlice {
		if visited[v] {
			return v, true
		}
		visited[v] = true
	}
	return "", false
}

func DelayedRetry(f func() error, maxRetries int, delay time.Duration) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = f()
		if err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("Reached max retries. Last error: %s", err.Error())
}
