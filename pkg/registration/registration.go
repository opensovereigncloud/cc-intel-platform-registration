package registration

import (
	"context"
	"time"

	mpmanagement "github.com/opensovereigncloud/cc-intel-platform-registration/internal/pkg/mp_management"
	intelservices "github.com/opensovereigncloud/cc-intel-platform-registration/pkg/intel_services"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/metrics"
	"go.uber.org/zap"
)

// RegistrationChecker is an interface to facilitate tests
type RegistrationChecker interface {
	Check() (metrics.StatusCodeMetric, error)
}

func NewRegistrationChecker(logger *zap.Logger) *DefaultRegistrationChecker {
	return &DefaultRegistrationChecker{
		log: logger,
	}
}

type DefaultRegistrationChecker struct {
	log *zap.Logger
}

func (rc *DefaultRegistrationChecker) Check() (metrics.StatusCodeMetric, error) {
	mp := mpmanagement.NewMPManagement()
	defer mp.Close()

	intelService := intelservices.NewIntelService(rc.log)

	isMachineRegistered, err := mp.IsMachineRegistered()
	if err != nil {
		rc.log.Error("unable to get the machine registration status", zap.Error(err))
		return metrics.StatusCodeMetric{Status: metrics.SgxUefiUnavailable}, err
	}

	if !isMachineRegistered {
		plaformManifest, platManErr := mp.GetPlatformManifest()
		if platManErr != nil {
			rc.log.Error("unable to get platform manifests ", zap.Error(platManErr))
			return metrics.StatusCodeMetric{Status: metrics.SgxUefiUnavailable}, platManErr
		}
		metric, regErr := intelService.RegisterPlatform(plaformManifest)

		// registration was successful
		if metric.Status == metrics.PlatformRebootNeeded {
			completeErr := mp.CompleteMachineRegistrationStatus()
			if completeErr != nil {
				rc.log.Error("unable to set registration status UEFI variable as complete ", zap.Error(completeErr))
				return metrics.StatusCodeMetric{Status: metrics.UefiPersistFailed}, completeErr
			}
		}
		return metric, regErr

	}

	// todo implement all cases here
	return metrics.StatusCodeMetric{Status: metrics.UnknownError}, nil
}

type RegistrationService struct {
	intervalDuration    time.Duration
	serverMetrics       *metrics.RegistrationServiceMetricsRegistry
	log                 *zap.Logger
	registrationChecker RegistrationChecker
}

func (r *RegistrationService) Run(ctx context.Context) error {
	err := r.serverMetrics.SetServiceStatusCodeToPending()

	if err != nil {
		return err
	}

	// first service check
	r.CheckRegistrationStatus()

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
	statusCodeMetric, err := r.registrationChecker.Check()
	if err != nil {
		r.log.Error("error getting the registration status", zap.Error(err))
	}
	r.log.Debug("Registration check completed", zap.String("status", statusCodeMetric.Status.String()))
	r.serverMetrics.UpdateServiceStatusCodeMetric(statusCodeMetric)
}

func NewRegistrationService(logger *zap.Logger, intervalDuration time.Duration) *RegistrationService {
	registrationService := &RegistrationService{
		serverMetrics:       metrics.NewRegistrationServiceMetricsRegistry(logger),
		registrationChecker: NewRegistrationChecker(logger),
		log:                 logger,
		intervalDuration:    intervalDuration,
	}

	return registrationService
}
