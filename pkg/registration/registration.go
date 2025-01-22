package registration

import (
	"context"
	"log/slog"
	"time"

	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/metrics"
)

// RegistrationChecker is an interface to facilitate tests
type RegistrationChecker interface {
	Check() metrics.StatusCode
}

type DefaultRegistrationChecker struct{}

func (rc *DefaultRegistrationChecker) Check() metrics.StatusCode {
	// todo implement all cases here
	return metrics.PENDING
}

type RegistrationService struct {
	intervalDuration    time.Duration
	serverMetrics       *metrics.RegistrationServiceMetricsRegistry
	registrationChecker RegistrationChecker
	log                 *slog.Logger
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
	status := r.registrationChecker.Check()
	r.log.Debug("Registration check completed", slog.String("status", status.String()))

	// todo(): update metrics here based on the status provided
}

func NewRegistrationService(logger *slog.Logger, intervalDuration time.Duration) *RegistrationService {
	registrationService := &RegistrationService{
		intervalDuration:    intervalDuration * time.Minute,
		serverMetrics:       metrics.NewRegistrationServiceMetricsRegistry(logger), //todo(): inject the logger into metrics registry
		registrationChecker: &DefaultRegistrationChecker{},
		log:                 logger,
	}
	return registrationService
}
