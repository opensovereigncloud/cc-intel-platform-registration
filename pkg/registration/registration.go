package registration

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/opensovereigncloud/cc-intel-platform-registration/internal/pkg/mp_management"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/metrics"
)

const INTEL_PLATFORM_REGISTRATION_ENDPOINT = "https://api.trustedservices.intel.com/sgx/registration/v1/platform"
const INTEL_REGISTRATION_REQUEST_TIMEOUT_IN_MINUTES = 2

// RegistrationChecker is an interface to facilitate tests
type RegistrationChecker interface {
	Check() (metrics.StatusCodeMetric, error)
}

func NewRegistrationChecker(logger *slog.Logger) *DefaultRegistrationChecker {
	return &DefaultRegistrationChecker{
		log: logger,
	}
}

type DefaultRegistrationChecker struct {
	log *slog.Logger
}

func (rc *DefaultRegistrationChecker) Check() (metrics.StatusCodeMetric, error) {
	mp, err := mp_management.NewMPManagement()

	if err != nil {
		log.Fatal(err)
	}
	defer mp.Close()

	machine_registration_status, err := mp.IsMachineRegistered()
	if err != nil {
		rc.log.Error("unable to get the machine registration status", slog.String("error", err.Error()))
		return metrics.StatusCodeMetric{Status: metrics.SGX_UEFI_UNAVAILABLE}, nil
	}

	if !machine_registration_status {
		plaform_manifest, err := mp.GetPlatformManifest()
		if err != nil {
			rc.log.Error("unable to get platform manifests ", slog.String("error", err.Error()))
			return metrics.StatusCodeMetric{Status: metrics.SGX_UEFI_UNAVAILABLE}, nil
		}
		metric, err := rc.registerPlatform(plaform_manifest)

		// registration was successful
		if metric.Status == metrics.PLATFORM_REBOOT_NEEDED {
			err := mp.CompleteMachineRegistrationStatus()
			if err != nil {
				rc.log.Error("unable to set registration status UEFI variable as complete ", slog.String("error", err.Error()))
				return metrics.StatusCodeMetric{Status: metrics.UEFI_PERSIST_FAILED}, nil
			}
		}
		return metric, err

	}

	// todo implement all cases here
	return metrics.StatusCodeMetric{Status: metrics.UNKNOWN_ERROR}, nil
}

func (r *DefaultRegistrationChecker) registerPlatform(platform_manifest mp_management.PlatformManifest) (metrics.StatusCodeMetric, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(INTEL_REGISTRATION_REQUEST_TIMEOUT_IN_MINUTES*time.Minute))
	defer cancel()
	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, INTEL_PLATFORM_REGISTRATION_ENDPOINT, bytes.NewReader(platform_manifest))
	if err != nil {
		r.log.Error("failed to create request", slog.String("error", err.Error()))
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	// Execute request
	resp, err := client.Do(req)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			r.log.Error("request timeout to Intel registration service", slog.String("error", err.Error()))
			return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("connection timeout: %w", err)
		}
		r.log.Error("failed to send request to Intel registration service", slog.String("error", err.Error()))
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		return metrics.StatusCodeMetric{Status: metrics.PLATFORM_REBOOT_NEEDED}, nil
	} else {
		error_code := resp.Header.Get("Error-Code")
		return metrics.CreateIntelStatusCodeMetric(resp.StatusCode, error_code), nil
	}

}

type RegistrationService struct {
	intervalDuration    time.Duration
	serverMetrics       *metrics.RegistrationServiceMetricsRegistry
	log                 *slog.Logger
	registrationChecker RegistrationChecker
}

func (r *RegistrationService) Run(ctx context.Context) error {
	err := r.serverMetrics.SetServiceStatusCodeToPending()

	if err != nil {
		return err
	}
	ticker := time.NewTicker(r.intervalDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.CheckRegistrationStatus()
		case <-ctx.Done():
			return nil
		}
	}
}

func (r *RegistrationService) CheckRegistrationStatus() {
	status_code_metric, err := r.registrationChecker.Check()
	if err != nil {
		r.log.Error("error getting the registration status", slog.String("err", err.Error()))
	}
	r.log.Debug("Registration check completed", slog.String("status", status_code_metric.Status.String()))
	r.serverMetrics.UpdateServiceStatusCodeMetric(status_code_metric)

}

func NewRegistrationService(logger *slog.Logger, intervalDuration time.Duration) *RegistrationService {
	registrationService := &RegistrationService{
		intervalDuration:    intervalDuration * time.Minute,
		serverMetrics:       metrics.NewRegistrationServiceMetricsRegistry(logger), //todo(): inject the logger into metrics registry
		registrationChecker: NewRegistrationChecker(logger),
		log:                 logger,
	}

	return registrationService
}
