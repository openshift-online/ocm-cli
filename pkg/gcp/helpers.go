package gcp

import (
	"fmt"
	"strconv"
)

const (
	monitoringBaseURL = "https://www.googleapis.com/auth/monitoring.read"
	promQLBaseURL     = "https://monitoring.googleapis.com/v1/projects/%s/location/global/prometheus/api/v1/query"
)

// PrometheusResponse represents the response from Prometheus API
// Example: {"status":"success","data":{"resultType":"vector","result":
// [{"metric":{"key":"value"},"value":[timestamp,"metric_value"]}]}}
type PrometheusResponse struct {
	Status string         `json:"status"`
	Data   PrometheusData `json:"data"`
}
type PrometheusData struct {
	ResultType string             `json:"resultType"`
	Result     []PrometheusResult `json:"result"`
}
type PrometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

func FmtSaResourceId(accountId, projectId string) string {
	return fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectId, accountId, projectId)
}

// GetFloatValue extracts the numeric value from the Value array
func (r *PrometheusResult) GetFloatValue() (float64, error) {
	if len(r.Value) < 2 {
		return 0, fmt.Errorf("invalid value format")
	}

	switch v := r.Value[1].(type) {
	case string:
		return strconv.ParseFloat(v, 64)
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected value type: %T", v)
	}
}
