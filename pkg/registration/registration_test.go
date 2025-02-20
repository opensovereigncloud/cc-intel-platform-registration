package registration

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
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

func (rc *TestRegistrationChecker) Check() metrics.StatusCode {
	if rc.counter == len(rc.metricSteps) {
		rc.counter = 0
	}
	currentMetric := rc.metricSteps[rc.counter]
	rc.counter++
	return currentMetric
}

func TestRegistrationServiceRun(t *testing.T) {

	observedZapCore, observedLogs := observer.New(zap.DebugLevel)
	observedLogger := zap.New(observedZapCore)
	logTest := slog.New(logr.ToSlogHandler(zapr.NewLogger(observedLogger)))
	metricsRegistry := metrics.NewRegistrationServiceMetricsRegistry(logTest)
	cases := []struct {
		msg                string
		metricSteps        []metrics.StatusCode
		expectedLogEntries []observer.LoggedEntry
	}{
		{
			msg: "first case example",
			metricSteps: []metrics.StatusCode{
				metrics.PLATFORM_DIRECTLY_REGISTERED,
				metrics.INTEL_CONNECT_FAILED,
				metrics.RETRY_NEEDED,
			},
			expectedLogEntries: []observer.LoggedEntry{
				{
					Entry: zapcore.Entry{
						Level:   zap.InfoLevel,
						Message: "Status code metric updated - Code: 0, HTTP StatusCode: 0, Intel Error code: ",
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
						Level:   zap.DebugLevel,
						Message: "Registration check completed",
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
			log:                 logTest,
		}

		testContext, cancelFunc := context.WithTimeout(context.TODO(), 100*time.Millisecond)
		defer cancelFunc()
		err := registrationService.Run(testContext)

		if err != nil {
			panic("registration failed: " + err.Error())
		}

	}
	c := cases[0]
	for i, observedLog := range observedLogs.All() {
		thisLogEntryEqualTo(t, c.expectedLogEntries[i], observedLog, c.msg)
	}

}

func thisLogEntryEqualTo(t testing.TB, this, other observer.LoggedEntry, msg string) {
	t.Helper()
	// todo(): also check .Data (which has the log fields)
	assert.Equal(t, this.Level, other.Level, msg)
	assert.Equal(t, this.Message, other.Message, msg)

}
