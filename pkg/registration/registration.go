package registration

import (
	"cc-intel-platform-registration/pkg/metrics"
	"context"
	"time"

	"github.com/sirupsen/logrus"
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
	log                 logrus.FieldLogger
}

func (r *RegistrationService) Run(ctx context.Context) error {
	r.serverMetrics.SetServiceStatusCodeToPending()
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
	r.log.WithField("status", status.String()).Debug("Registration check completed")
	// todo(): update metrics here based on the status provided
}

func NewRegistrationService(intervalDuration time.Duration) *RegistrationService {
	logger := logrus.StandardLogger()
	registrationService := &RegistrationService{
		intervalDuration:    intervalDuration * time.Minute,
		serverMetrics:       metrics.NewRegistrationServiceMetricsRegistry(logger), //todo(): inject the logger into metrics registry
		registrationChecker: &DefaultRegistrationChecker{},
		log:                 logger,
	}
	return registrationService
}
