package metrics

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestGetDetails(t *testing.T) {
	cases := []struct {
		msg            string
		statusCode     StatusCode
		wantedDetails  StatusCodeDetails
		wantedIntValue int
	}{
		{
			msg:        "Pending returns the expected details",
			statusCode: Pending,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 0,
		},
		{
			msg:        "SgxUefiUnavailable returns the expected details",
			statusCode: SgxUefiUnavailable,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 1,
		},
		{
			msg:        "RetryNeeded returns the expected details",
			statusCode: RetryNeeded,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 2,
		},
		{
			msg:        "SgxResetNeeded returns the expected details",
			statusCode: SgxResetNeeded,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: true,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 3,
		},
		{
			msg:        "UefiPersistFailed returns the expected details",
			statusCode: UefiPersistFailed,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 4,
		},
		{
			msg:        "PlatformRebootNeeded returns the expected details",
			statusCode: PlatformRebootNeeded,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 5,
		},
		{
			msg:        "PlatformDirectlyRegistered returns the expected details",
			statusCode: PlatformDirectlyRegistered,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 9,
		},
		{
			msg:        "IntelConnectFailed returns the expected details",
			statusCode: IntelConnectFailed,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 10,
		},
		{
			msg:        "InvalidRegistrationRequest returns the expected details",
			statusCode: InvalidRegistrationRequest,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: true,
				RequiresIntelErrCode:   true,
			},
			wantedIntValue: 11,
		},
		{
			msg:        "IntelRegServiceRequestFailed returns the expected details",
			statusCode: IntelRegServiceRequestFailed,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: true,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 12,
		},
		{
			msg:        "UnknownError returns the expected details",
			statusCode: UnknownError,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				RequiresIntelErrCode:   false,
			},
			wantedIntValue: 99,
		},
	}

	for _, c := range cases {
		actualDetails := c.statusCode.GetDetails()
		assert.Equal(t, c.wantedDetails, actualDetails, c.msg)
		assert.Equal(t, c.wantedIntValue, int(c.statusCode), c.msg)
	}
}

func TestGetStatusCodeString(t *testing.T) {
	cases := []struct {
		msg          string
		statusCode   StatusCode
		wantedString string
	}{
		{
			msg:          "Pending returns the expected details",
			statusCode:   Pending,
			wantedString: "Pending: pending execution",
		},
		{
			msg:          "SgxUefiUnavailable returns the expected details",
			statusCode:   SgxUefiUnavailable,
			wantedString: "SgxUefiUnavailable: SGX UEFI variables not available",
		},
		{
			msg:          "RetryNeeded returns the expected details",
			statusCode:   RetryNeeded,
			wantedString: "RetryNeeded: impossible to determine the registration status; please reattempt",
		},
		{
			msg:          "SgxResetNeeded returns the expected details",
			statusCode:   SgxResetNeeded,
			wantedString: "SgxResetNeeded: impossible to determine the registration status; please reset the SGX",
		},
		{
			msg:          "UefiPersistFailed returns the expected details",
			statusCode:   UefiPersistFailed,
			wantedString: "UefiPersistFailed: failed to persist the UEFI variable content",
		},
		{
			msg:          "PlatformRebootNeeded returns the expected details",
			statusCode:   PlatformRebootNeeded,
			wantedString: "PlatformRebootNeeded: platform registered successfully and a reboot is required",
		},
		{
			msg:          "PlatformDirectlyRegistered returns the expected details",
			statusCode:   PlatformDirectlyRegistered,
			wantedString: "PlatformDirectlyRegistered: platform directly registered",
		},
		{
			msg:          "IntelConnectFailed returns the expected details",
			statusCode:   IntelConnectFailed,
			wantedString: "IntelConnectFailed: failed to connect to Intel RS",
		},
		{
			msg:          "InvalidRegistrationRequest returns the expected details",
			statusCode:   InvalidRegistrationRequest,
			wantedString: "InvalidRegistrationRequest: invalid registration request",
		},
		{
			msg:          "IntelRegServiceRequestFailed returns the expected details",
			statusCode:   IntelRegServiceRequestFailed,
			wantedString: "IntelRegServiceRequestFailed: intel RS could not process the request",
		},
		{
			msg:          "UnknownError returns the expected details",
			statusCode:   UnknownError,
			wantedString: "UnknownError",
		},
	}

	for _, c := range cases {
		actualDetails := c.statusCode.String()
		assert.Equal(t, c.wantedString, actualDetails, c.msg)
	}
}

func TestUpdateServiceStatusCodeMetricWarning(t *testing.T) {

	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)
	metricsRegistry := NewRegistrationServiceMetricsRegistry(observedLogger)
	// logTest.
	cases := []struct {
		msg                string
		metricUpdate       StatusCodeMetric
		expectedLogEntries []observer.LoggedEntry
		expectError        bool
	}{
		{
			msg: "pending state has no warning",
			metricUpdate: StatusCodeMetric{

				Status:         Pending,
				HttpStatusCode: "",
				IntelError:     "",
			},
			expectError: false,

			expectedLogEntries: []observer.LoggedEntry{
				{
					Entry: zapcore.Entry{
						Level:   zap.InfoLevel,
						Message: fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s", Pending, "", ""),
					},
				},
			},
		},
		{
			msg: "IntelRegServiceRequestFailed requires http code label ",
			metricUpdate: StatusCodeMetric{
				Status:         IntelRegServiceRequestFailed,
				HttpStatusCode: "",
				IntelError:     "",
			},

			expectError: true,
			expectedLogEntries: []observer.LoggedEntry{
				{
					Entry: zapcore.Entry{
						Level:   zap.InfoLevel,
						Message: fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s", Pending, "", ""),
					},
				},
			},
		},
		{
			msg: "SgxResetNeeded requires http code label ",
			metricUpdate: StatusCodeMetric{
				Status:         SgxResetNeeded,
				HttpStatusCode: "",
				IntelError:     "",
			},

			expectError:        true,
			expectedLogEntries: []observer.LoggedEntry{},
		},
		{
			msg: "InvalidRegistrationRequest requires http code label ",
			metricUpdate: StatusCodeMetric{
				Status:         InvalidRegistrationRequest,
				HttpStatusCode: "",
				IntelError:     "",
			},

			expectError:        true,
			expectedLogEntries: []observer.LoggedEntry{},
		},
		{
			msg: "InvalidRegistrationRequest requires http code label and intel error code ",
			metricUpdate: StatusCodeMetric{
				Status:         InvalidRegistrationRequest,
				HttpStatusCode: "400",
				IntelError:     "",
			},

			expectError:        true,
			expectedLogEntries: []observer.LoggedEntry{},
		},
		{
			msg: "InvalidRegistrationRequest requires http code label and intel error code ",
			metricUpdate: StatusCodeMetric{
				Status:         InvalidRegistrationRequest,
				HttpStatusCode: "400",
				IntelError:     "InvalidRequest",
			},

			expectError: false,
			expectedLogEntries: []observer.LoggedEntry{
				{
					Entry: zapcore.Entry{
						Level:   zap.InfoLevel,
						Message: fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s", InvalidRegistrationRequest, "400", "InvalidRequest"),
					},
				},
			},
		},
		{
			msg: "IntelRegServiceRequestFailed with http status code works fine",
			metricUpdate: StatusCodeMetric{

				Status:         IntelRegServiceRequestFailed,
				HttpStatusCode: "400",
				IntelError:     "",
			},
			expectError: false,
			expectedLogEntries: []observer.LoggedEntry{
				{
					Entry: zapcore.Entry{
						Level:   zap.InfoLevel,
						Message: fmt.Sprintf("Status code metric updated - Code: %d, HTTP StatusCode: %s, Intel Error code: %s", IntelRegServiceRequestFailed, "400", ""),
					},
				},
			},
		},
	}
	for _, c := range cases {
		err := metricsRegistry.UpdateServiceStatusCodeMetric(c.metricUpdate)
		if c.expectError {
			assert.Error(t, err, c.msg)
		}
		for _, expectedEntry := range c.expectedLogEntries {
			thisLogEntryEqualTo(t, expectedEntry, observedLogs.All()[observedLogs.Len()-1], c.msg)
		}
	}

}

func thisLogEntryEqualTo(t testing.TB, this, other observer.LoggedEntry, msg string) {
	t.Helper()
	assert.Equal(t, this.Level, other.Level, msg)
	assert.Equal(t, this.Message, other.Message, msg)

}
