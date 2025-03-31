package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

const (
	// metrics definitions
	RegistrationServiceStatusCodeMetricValue  = "service_status_code"
	RegistrationServicePanicCountsMetricValue = "application_panics_total"

	// label definitions
	HttpStatusCodeLabel = "http_status_code"
	IntelErrorCodeLabel = "intel_error_code"
)

// Define a custom type for status codes
type StatusCode int

type StatusCodeDetails struct {
	RequiresHTTPStatusCode bool
	RequiresIntelErrCode   bool
}

const (
	Pending                      StatusCode = iota // 0
	SgxUefiUnavailable           StatusCode = 1
	RetryNeeded                  StatusCode = 2
	SgxResetNeeded               StatusCode = 3
	UefiPersistFailed            StatusCode = 4
	PlatformRebootNeeded         StatusCode = 5
	PlatformDirectlyRegistered   StatusCode = 9
	IntelConnectFailed           StatusCode = 10
	InvalidRegistrationRequest   StatusCode = 11
	IntelRegServiceRequestFailed StatusCode = 12
	UnknownError                 StatusCode = 99
)

func (s StatusCode) GetDetails() StatusCodeDetails {
	switch s {
	case InvalidRegistrationRequest:
		return StatusCodeDetails{
			RequiresHTTPStatusCode: true,
			RequiresIntelErrCode:   true,
		}
	case IntelRegServiceRequestFailed, SgxResetNeeded:
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
	case Pending:
		return "Pending: pending execution"
	case SgxUefiUnavailable:
		return "SgxUefiUnavailable: SGX UEFI variables not available"
	case RetryNeeded:
		return "RetryNeeded: impossible to determine the registration status; please reattempt"
	case SgxResetNeeded:
		return "SgxResetNeeded: impossible to determine the registration status; please reset the SGX"
	case PlatformRebootNeeded:
		return "PlatformRebootNeeded: platform registered successfully and a reboot is required"
	case UefiPersistFailed:
		return "UefiPersistFailed: failed to persist the UEFI variable content"
	case PlatformDirectlyRegistered:
		return "PlatformDirectlyRegistered: platform directly registered"
	case IntelConnectFailed:
		return "IntelConnectFailed: failed to connect to Intel RS"
	case InvalidRegistrationRequest:
		return "InvalidRegistrationRequest: invalid registration request"
	case IntelRegServiceRequestFailed:
		return "IntelRegServiceRequestFailed: intel RS could not process the request"
	default:
		return "UnknownError"
	}
}

type StatusCodeMetric struct {
	Status         StatusCode
	HttpStatusCode string
	IntelError     string
}

func CreateUnknownErrorStatusCodeMetric() StatusCodeMetric {
	return StatusCodeMetric{
		Status: UnknownError,
	}

}

type RegistrationServiceMetricsRegistry struct {
	log *zap.Logger
}

func NewRegistrationServiceMetricsRegistry(logger *zap.Logger) *RegistrationServiceMetricsRegistry {
	return &RegistrationServiceMetricsRegistry{
		log: logger,
	}
}

var (
	RegistrationServiceStatusCodeMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: RegistrationServiceStatusCodeMetricValue,
			Help: "Current status code of the registration service",
		},
		[]string{HttpStatusCodeLabel, IntelErrorCodeLabel},
	)

	RegistrationServicePanicCountsMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: RegistrationServicePanicCountsMetricValue,
		Help: "Total number of go routines panics",
	})
)

// helper function to service status code to pending
func IncrementPanicCounts() {
	RegistrationServicePanicCountsMetric.Inc()
}

// helper function to service status code to pending
func (s *RegistrationServiceMetricsRegistry) SetServiceStatusCodeToPending() error {
	metricValue := StatusCodeMetric{
		Status:         Pending,
		HttpStatusCode: "",
		IntelError:     "",
	}
	return s.UpdateServiceStatusCodeMetric(metricValue)
}

func (s *RegistrationServiceMetricsRegistry) UpdateServiceStatusCodeMetric(metricValue StatusCodeMetric) error {
	// Validate required labels
	statusDetails := metricValue.Status.GetDetails()
	if statusDetails.RequiresHTTPStatusCode && metricValue.HttpStatusCode == "" {
		return fmt.Errorf("warning: status code %d requires HTTP status code but none provided",
			metricValue.Status)
	}

	if statusDetails.RequiresIntelErrCode && metricValue.IntelError == "" {
		return fmt.Errorf("warning: Status code %d requires Intel Error code but none provided",
			metricValue.Status)
	}
	// Set the new metric value with labels
	RegistrationServiceStatusCodeMetric.With(prometheus.Labels{
		HttpStatusCodeLabel: metricValue.HttpStatusCode,
		IntelErrorCodeLabel: metricValue.IntelError,
	}).Set(float64(metricValue.Status))

	s.log.Info(
		fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s",
			metricValue.Status, metricValue.HttpStatusCode, metricValue.IntelError),
		zap.Int(RegistrationServiceStatusCodeMetricValue, metricValue.Status.toInt()),
		zap.String(HttpStatusCodeLabel, metricValue.HttpStatusCode),
		zap.String(IntelErrorCodeLabel, metricValue.IntelError))

	return nil

}
