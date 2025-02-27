package metrics

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// metrics definitions
	SERVICE_STATUS_CODE_METRIC = "service_status_code"

	// label definitions
	HTTP_STATUS_CODE_LABEL = "http_status_code"
	INTEL_ERROR_CODE_LABEL = "intel_error_code"
)

// Define a custom type for status codes
type StatusCode int

type StatusCodeDetails struct {
	RequiresHTTPStatusCode bool
	RequiresIntelErrCode   bool
}

const (
	PENDING                      StatusCode = iota // 0
	SGX_UEFI_UNAVAILABLE         StatusCode = 1
	RETRY_NEEDED                 StatusCode = 2
	SGX_RESET_NEEDED             StatusCode = 3
	UEFI_PERSIST_FAILED          StatusCode = 4
	PLATFORM_REBOOT_NEEDED       StatusCode = 5
	PLATFORM_DIRECTLY_REGISTERED StatusCode = 9
	INTEL_CONNECT_FAILED         StatusCode = 10
	INVALID_REGISTRATION_REQUEST StatusCode = 11
	INTEL_RS_REQUEST_FAILED      StatusCode = 12
	UNKNOWN_ERROR                StatusCode = 99
)

func (s StatusCode) GetDetails() StatusCodeDetails {
	switch s {
	case INVALID_REGISTRATION_REQUEST:
		return StatusCodeDetails{
			RequiresHTTPStatusCode: true,
			RequiresIntelErrCode:   true,
		}
	case INTEL_RS_REQUEST_FAILED, SGX_RESET_NEEDED:
		return StatusCodeDetails{
			RequiresHTTPStatusCode: true,
			RequiresIntelErrCode:   false,
		}
	default:
		return StatusCodeDetails{
			RequiresHTTPStatusCode: false,
			RequiresIntelErrCode:   false,
		}
	}

}

func (s StatusCode) toInt() int {
	return int(s)
}

// Add a String() method for easy conversion to string
func (s StatusCode) String() string {
	switch s {
	case PENDING:
		return "PENDING: pending execution"
	case SGX_UEFI_UNAVAILABLE:
		return "SGX_UEFI_UNAVAILABLE: SGX UEFI variables not available"
	case RETRY_NEEDED:
		return "RETRY_NEEDED: impossible to determine the registration status; please reattempt"
	case SGX_RESET_NEEDED:
		return "SGX_RESET_NEEDED: impossible to determine the registration status; please reset the SGX"
	case PLATFORM_REBOOT_NEEDED:
		return "PLATFORM_REBOOT_NEEDED: platform registered successfully and a reboot is required"
	case UEFI_PERSIST_FAILED:
		return "UEFI_PERSIST_FAILED: failed to persist the UEFI variable content"
	case PLATFORM_DIRECTLY_REGISTERED:
		return "PLATFORM_DIRECTLY_REGISTERED: platform directly registered"
	case INTEL_CONNECT_FAILED:
		return "INTEL_CONNECT_FAILED: failed to connect to Intel RS"
	case INVALID_REGISTRATION_REQUEST:
		return "INVALID_REGISTRATION_REQUEST: invalid registration request"
	case INTEL_RS_REQUEST_FAILED:
		return "INTEL_RS_REQUEST_FAILED: intel RS could not process the request"
	default:
		return "UNKNOWN_ERROR"
	}
}

type StatusCodeMetric struct {
	Status         StatusCode
	HttpStatusCode string
	IntelError     string
}

func CreateIntelStatusCodeMetricForPlatformRegistration(http_status_code int, intel_error_code string) StatusCodeMetric {
	var Status StatusCode
	if http_status_code >= 400 && http_status_code < 500 {
		Status = INVALID_REGISTRATION_REQUEST
	} else {
		Status = INTEL_RS_REQUEST_FAILED
	}
	return StatusCodeMetric{
		Status:         Status,
		HttpStatusCode: strconv.Itoa(http_status_code),
		IntelError:     intel_error_code,
	}
}

func CreateIntelStatusCodeMetricForDirectRegistration(http_status_code int, intel_error_code string) StatusCodeMetric {

	var Status StatusCode
	if http_status_code == 404 {
		Status = SGX_RESET_NEEDED
	} else {
		Status = RETRY_NEEDED
	}

	return StatusCodeMetric{
		Status:         Status,
		HttpStatusCode: strconv.Itoa(http_status_code),
		IntelError:     intel_error_code,
	}
}

func CreateUnknownErrorStatusCodeMetric() StatusCodeMetric {
	return StatusCodeMetric{
		Status: UNKNOWN_ERROR,
	}
	// error_code
}

type RegistrationServiceMetricsRegistry struct {
	metrics map[string]prometheus.Collector
	log     *slog.Logger
}

func NewRegistrationServiceMetricsRegistry(logger *slog.Logger) *RegistrationServiceMetricsRegistry {
	return &RegistrationServiceMetricsRegistry{
		metrics: map[string]prometheus.Collector{
			SERVICE_STATUS_CODE_METRIC: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: SERVICE_STATUS_CODE_METRIC,
					Help: "Current status code of the registration service",
				},
				[]string{HTTP_STATUS_CODE_LABEL, INTEL_ERROR_CODE_LABEL},
			),
		},
		log: logger,
	}
}

// helper function to service status code to pending
func (s *RegistrationServiceMetricsRegistry) SetServiceStatusCodeToPending() error {
	metric_value := StatusCodeMetric{
		Status:         PENDING,
		HttpStatusCode: "",
		IntelError:     "",
	}
	return s.UpdateServiceStatusCodeMetric(metric_value)
}

func (s *RegistrationServiceMetricsRegistry) UpdateServiceStatusCodeMetric(metric_value StatusCodeMetric) error {
	// Validate required labels
	status_details := metric_value.Status.GetDetails()
	if status_details.RequiresHTTPStatusCode && metric_value.HttpStatusCode == "" {
		return fmt.Errorf("warning: Status code %d requires HTTP status code but none provided",
			metric_value.Status)
	}

	if status_details.RequiresIntelErrCode && metric_value.IntelError == "" {
		return fmt.Errorf("warning: Status code %d requires Intel Error code but none provided",
			metric_value.Status)
	}
	// Set the new metric value with labels
	if c, ok := s.metrics[SERVICE_STATUS_CODE_METRIC].(*prometheus.GaugeVec); ok {
		c.With(prometheus.Labels{
			"http_status_code": metric_value.HttpStatusCode,
			"intel_error_code": metric_value.IntelError,
		}).Set(float64(metric_value.Status))
		s.log.Info(
			fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s",
				metric_value.Status, metric_value.HttpStatusCode, metric_value.IntelError),
			slog.Int(SERVICE_STATUS_CODE_METRIC, metric_value.Status.toInt()),
			slog.String(HTTP_STATUS_CODE_LABEL, metric_value.HttpStatusCode),
			slog.String(INTEL_ERROR_CODE_LABEL, metric_value.IntelError))
	} else {
		return fmt.Errorf("internal Error: metric %s was not registered in metric registry", SERVICE_STATUS_CODE_METRIC)
	}
	return nil

}
