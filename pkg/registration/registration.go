package registration

import (
	"context"
	"fmt"
	"time"

	mpmanagement "github.com/opensovereigncloud/cc-intel-platform-registration/internal/pkg/mp_management"
	sgxplatforminfo "github.com/opensovereigncloud/cc-intel-platform-registration/internal/pkg/sgx_platform_info"
	config "github.com/opensovereigncloud/cc-intel-platform-registration/pkg/config"
	intelservices "github.com/opensovereigncloud/cc-intel-platform-registration/pkg/intel_services"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/metrics"
	"go.uber.org/zap"
)

// RegistrationChecker is an interface to facilitate tests
type RegistrationChecker interface {
	Check() (metrics.StatusCodeMetric, error)
}

func NewRegistrationChecker(logger *zap.Logger, cfg *config.RegistrationServiceConfig, metricsRegistry *metrics.RegistrationServiceMetricsRegistry) *DefaultRegistrationChecker {
	return &DefaultRegistrationChecker{
		log:              logger,
		regServiceConfig: cfg,
		metricsRegistry:  metricsRegistry,
	}
}

type DefaultRegistrationChecker struct {
	log              *zap.Logger
	regServiceConfig *config.RegistrationServiceConfig
	metricsRegistry  *metrics.RegistrationServiceMetricsRegistry
}

func (rc *DefaultRegistrationChecker) Check() (metrics.StatusCodeMetric, error) {
	mp := mpmanagement.NewMPManagement()
	defer mp.Close()

	intelService, err := intelservices.NewIntelService(rc.log, rc.regServiceConfig)
	if err != nil {
		return metrics.StatusCodeMetric{Status: metrics.UnknownError},
			fmt.Errorf("failed to create intel service: %w", err)
	}

	isMachineRegistered, err := mp.IsMachineRegistered()
	if err != nil {
		return metrics.StatusCodeMetric{Status: metrics.SgxUefiUnavailable}, err
	}

	if !isMachineRegistered {
		plaformManifest, platManErr := mp.GetPlatformManifest()
		if platManErr != nil {
			return metrics.StatusCodeMetric{Status: metrics.SgxUefiUnavailable}, platManErr
		}
		// Pass metrics registry to RegisterPlatform
		metric, regErr := intelService.RegisterPlatform(plaformManifest, rc.metricsRegistry)

		// registration was successful
		if metric.Status == metrics.PlatformRebootNeeded {
			completeErr := mp.CompleteMachineRegistrationStatus()
			if completeErr != nil {
				return metrics.StatusCodeMetric{Status: metrics.UefiPersistFailed}, completeErr
			}
		}
		return metric, regErr
	}

	platformInfo, err := sgxplatforminfo.GetSgxPlatformInfo()
	if err != nil {
		return metrics.StatusCodeMetric{Status: metrics.RetryNeeded}, err
	}

	// Pass metrics registry to RetrievePCK
	metric, err := intelService.RetrievePCK(platformInfo, rc.metricsRegistry)
	return metric, err
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

	// first check
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
		r.log.Error("unable to get the registration status", zap.Error(err))
	}
	r.log.Debug("Registration check completed", zap.String("status", statusCodeMetric.Status.String()))
	err = r.serverMetrics.UpdateServiceStatusCodeMetric(statusCodeMetric)
	if err != nil {
		r.log.Error("unable to update registration service status code metric", zap.Error(err))
	}
}

func NewRegistrationService(logger *zap.Logger, cfg *config.RegistrationServiceConfig, intervalDuration time.Duration) *RegistrationService {
	metricsRegistry := metrics.NewRegistrationServiceMetricsRegistry(logger)

	registrationService := &RegistrationService{
		serverMetrics:       metricsRegistry,
		registrationChecker: NewRegistrationChecker(logger, cfg, metricsRegistry),
		log:                 logger,
		intervalDuration:    intervalDuration,
	}

	return registrationService
}
