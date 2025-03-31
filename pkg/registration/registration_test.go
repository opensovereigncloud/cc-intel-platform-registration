package registration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/metrics"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/stretchr/testify/assert"
)

type TestRegistrationChecker struct {
	metricSteps []metrics.StatusCode
	counter     int
}

func (rc *TestRegistrationChecker) Check() (metrics.StatusCodeMetric, error) {
	if rc.counter == len(rc.metricSteps) {
		rc.counter = 0
	}
	currentMetric := rc.metricSteps[rc.counter]
	rc.counter++
	return metrics.StatusCodeMetric{Status: currentMetric}, nil
}

func TestRegistrationServiceRun(t *testing.T) {

	observedZapCore, observedLogs := observer.New(zap.DebugLevel)
	observedLogger := zap.New(observedZapCore)
	metricsRegistry := metrics.NewRegistrationServiceMetricsRegistry(observedLogger)
	cases := []struct {
		msg                string
		metricSteps        []metrics.StatusCode
		expectedLogEntries []observer.LoggedEntry
	}{
		{
			msg: "first case example",
			metricSteps: []metrics.StatusCode{
				metrics.PlatformDirectlyRegistered,
				metrics.IntelConnectFailed,
				metrics.RetryNeeded,
			},
			expectedLogEntries: []observer.LoggedEntry{
				{
					Entry: zapcore.Entry{
						Level:   zap.InfoLevel,
						Message: fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s", metrics.Pending, "", ""),
					},
				},
				{
					Entry: zapcore.Entry{
						Level:   zap.DebugLevel,
						Message: "Registration check completed",
					},
				},

				{
					Entry: zapcore.Entry{
						Level:   zap.InfoLevel,
						Message: fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s", metrics.PlatformDirectlyRegistered, "", ""),
					},
				},
				{
					Entry: zapcore.Entry{
						Level:   zap.DebugLevel,
						Message: "Registration check completed",
					},
				},
				{
					Entry: zapcore.Entry{
						Level:   zap.InfoLevel,
						Message: fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s", metrics.IntelConnectFailed, "", ""),
					},
				},
				{
					Entry: zapcore.Entry{
						Level:   zap.DebugLevel,
						Message: "Registration check completed",
					},
				},
				{
					Entry: zapcore.Entry{
						Level:   zap.InfoLevel,
						Message: fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s", metrics.RetryNeeded, "", ""),
					},
				},
				{
					Entry: zapcore.Entry{
						Level:   zap.DebugLevel,
						Message: "Registration check completed",
					},
				},
			},
		},
	}
	for _, c := range cases {
		testRegistrationChecker := &TestRegistrationChecker{
			metricSteps: c.metricSteps,
		}

		registrationService := &RegistrationService{
			intervalDuration:    1 * time.Millisecond,
			serverMetrics:       metricsRegistry,
			registrationChecker: testRegistrationChecker,
			log:                 observedLogger,
		}

		testContext, cancelFunc := context.WithTimeout(context.TODO(), 100*time.Millisecond)
		defer cancelFunc()
		err := registrationService.Run(testContext)

		if err != nil {
			panic("registration failed: " + err.Error())
		}

	}
	c := cases[0]
	for i, expectedLog := range c.expectedLogEntries {
		observedLog := observedLogs.All()[i]
		thisLogEntryEqualTo(t, expectedLog, observedLog, c.msg)
	}

}

func thisLogEntryEqualTo(t testing.TB, this, other observer.LoggedEntry, msg string) {
	t.Helper()
	assert.Equal(t, this.Level, other.Level, msg)
	assert.Equal(t, this.Message, other.Message, msg)

}
