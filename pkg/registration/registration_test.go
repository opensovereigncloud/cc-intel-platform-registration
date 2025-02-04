package registration

import (
	"cc-intel-platform-registration/pkg/metrics"
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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
	logTest, logHook := test.NewNullLogger()
	logTest.SetLevel(logrus.DebugLevel)
	metricsRegistry := metrics.NewRegistrationServiceMetricsRegistry(logTest)
	cases := []struct {
		msg                string
		metricSteps        []metrics.StatusCode
		expectedLogEntries []logrus.Entry
	}{
		{
			msg: "first case example",
			metricSteps: []metrics.StatusCode{
				metrics.PLATFORM_DIRECTLY_REGISTERED,
				metrics.INTEL_CONNECT_FAILED,
				metrics.RETRY_NEEDED,
			},
			expectedLogEntries: []logrus.Entry{
				{
					Level:   logrus.InfoLevel,
					Message: "Status code metric updated - Code: 0, HTTP StatusCode: 0, Intel Error code: ",
					Data: logrus.Fields{
						metrics.SERVICE_STATUS_CODE_METRIC: metrics.PENDING,
						metrics.HTTP_STATUS_CODE_LABEL:     "0",
						metrics.INTEL_ERROR_CODE_LABEL:     "",
					},
				},
				{
					Level:   logrus.DebugLevel,
					Message: "Registration check completed",
					Data: logrus.Fields{
						"status": metrics.PLATFORM_DIRECTLY_REGISTERED.String(),
					},
				},
				{
					Level:   logrus.DebugLevel,
					Message: "Registration check completed",
					Data: logrus.Fields{
						"status": metrics.INTEL_CONNECT_FAILED.String(),
					},
				},
				{
					Level:   logrus.DebugLevel,
					Message: "Registration check completed",
					Data: logrus.Fields{
						"status": metrics.RETRY_NEEDED.String(),
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
		registrationService.Run(testContext)

		for i, expectedEntry := range c.expectedLogEntries {
			println(expectedEntry.Message)
			thisLogEntryEqualTo(t, expectedEntry, logHook.Entries[i], c.msg)
			println("not cool")

		}
	}
}

func thisLogEntryEqualTo(t testing.TB, this, other logrus.Entry, msg string) {
	t.Helper()
	// todo(): also check .Data (which has the log fields)
	assert.Equal(t, this.Level, other.Level, msg)
	assert.Equal(t, this.Message, other.Message, msg)
	assert.Equal(t, this.Data, other.Data, msg)
}
