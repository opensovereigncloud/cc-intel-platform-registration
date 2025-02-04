package metrics

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestGetDetails(t *testing.T) {
	cases := []struct {
		msg            string
		statusCode     StatusCode
		wantedDetails  StatusCodeDetails
		wantedIntValue int
	}{
		{
			msg:        "PENDING returns the expected details",
			statusCode: PENDING,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 0,
		},
		{
			msg:        "SGX_UEFI_UNAVAILABLE returns the expected details",
			statusCode: SGX_UEFI_UNAVAILABLE,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 1,
		},
		{
			msg:        "RETRY_NEEDED returns the expected details",
			statusCode: RETRY_NEEDED,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 2,
		},
		{
			msg:        "SGX_RESET_NEEDED returns the expected details",
			statusCode: SGX_RESET_NEEDED,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: true,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 3,
		},
		{
			msg:        "UEFI_PERSIST_FAILED returns the expected details",
			statusCode: UEFI_PERSIST_FAILED,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 4,
		},
		{
			msg:        "REBOOT_NEEDED returns the expected details",
			statusCode: PLATFORM_REBOOT_NEEDED,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 5,
		},
		{
			msg:        "DIRECT_REGISTERED returns the expected details",
			statusCode: PLATFORM_DIRECTLY_REGISTERED,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 9,
		},
		{
			msg:        "INTEL_CONNECT_FAILED returns the expected details",
			statusCode: INTEL_CONNECT_FAILED,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 10,
		},
		{
			msg:        "INVALID_REGISTRATION_REQUEST returns the expected details",
			statusCode: INVALID_REGISTRATION_REQUEST,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: true,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 11,
		},
		{
			msg:        "INTEL_RS_REQUEST_FAILED returns the expected details",
			statusCode: INTEL_RS_REQUEST_FAILED,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: true,
				AllowsIntelErrCode:     false,
			},
			wantedIntValue: 12,
		},
		{
			msg:        "UNKNOWN_ERROR returns the expected details",
			statusCode: UNKNOWN_ERROR,
			wantedDetails: StatusCodeDetails{
				RequiresHTTPStatusCode: false,
				AllowsIntelErrCode:     false,
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
			msg:          "PENDING returns the expected details",
			statusCode:   PENDING,
			wantedString: "PENDING: pending execution",
		},
		{
			msg:          "SGX_UEFI_UNAVAILABLE returns the expected details",
			statusCode:   SGX_UEFI_UNAVAILABLE,
			wantedString: "SGX_UEFI_UNAVAILABLE: SGX UEFI variables not available",
		},
		{
			msg:          "RETRY_NEEDED returns the expected details",
			statusCode:   RETRY_NEEDED,
			wantedString: "RETRY_NEEDED: impossible to determine the registration status; please reattempt",
		},
		{
			msg:          "SGX_RESET_NEEDED returns the expected details",
			statusCode:   SGX_RESET_NEEDED,
			wantedString: "SGX_RESET_NEEDED: impossible to determine the registration status; please reset the SGX",
		},
		{
			msg:          "UEFI_PERSIST_FAILED returns the expected details",
			statusCode:   UEFI_PERSIST_FAILED,
			wantedString: "UEFI_PERSIST_FAILED: failed to persist the UEFI variable content",
		},
		{
			msg:          "REBOOT_NEEDED returns the expected details",
			statusCode:   PLATFORM_REBOOT_NEEDED,
			wantedString: "PLATFORM_REBOOT_NEEDED: platform registered successfully and a reboot is required",
		},
		{
			msg:          "DIRECT_REGISTERED returns the expected details",
			statusCode:   PLATFORM_DIRECTLY_REGISTERED,
			wantedString: "PLATFORM_DIRECTLY_REGISTERED: platform directly registered",
		},
		{
			msg:          "INTEL_CONNECT_FAILED returns the expected details",
			statusCode:   INTEL_CONNECT_FAILED,
			wantedString: "INTEL_CONNECT_FAILED: failed to connect to Intel RS",
		},
		{
			msg:          "INVALID_REGISTRATION_REQUEST returns the expected details",
			statusCode:   INVALID_REGISTRATION_REQUEST,
			wantedString: "INVALID_REGISTRATION_REQUEST: invalid registration request",
		},
		{
			msg:          "INTEL_RS_REQUEST_FAILED returns the expected details",
			statusCode:   INTEL_RS_REQUEST_FAILED,
			wantedString: "INTEL_RS_REQUEST_FAILED: intel RS could not process the request",
		},
		{
			msg:          "UNKNOWN_ERROR returns the expected details",
			statusCode:   UNKNOWN_ERROR,
			wantedString: "UNKNOWN_ERROR",
		},
	}

	for _, c := range cases {
		actualDetails := c.statusCode.String()
		assert.Equal(t, c.wantedString, actualDetails, c.msg)
	}
}

func TestUpdateServiceStatusCodeMetricWarning(t *testing.T) {
	logTest, logHook := test.NewNullLogger()
	logTest.SetLevel(logrus.DebugLevel)
	metricsRegistry := NewRegistrationServiceMetricsRegistry(logTest)
	cases := []struct {
		msg                string
		metricUpdate       StatusCodeMetric
		expectedLogEntries []logrus.Entry
		expectError        bool
	}{
		{
			msg: "pending state has no warning",
			metricUpdate: StatusCodeMetric{

				Status:         PENDING,
				HttpStatusCode: "",
				IntelError:     "",
			},
			expectError: false,

			expectedLogEntries: []logrus.Entry{
				{
					Level:   logrus.InfoLevel,
					Message: "Status code metric updated - Code: 0, HTTP StatusCode: , Intel Error code: ",
					Data: logrus.Fields{
						SERVICE_STATUS_CODE_METRIC: PENDING,
						HTTP_STATUS_CODE_LABEL:     "",
						INTEL_ERROR_CODE_LABEL:     "",
					},
				},
			},
		},
		{
			msg: "INTEL_RS_REQUEST_FAILED requires http code label ",
			metricUpdate: StatusCodeMetric{
				Status:         INTEL_RS_REQUEST_FAILED,
				HttpStatusCode: "",
				IntelError:     "",
			},

			expectError: true,
			expectedLogEntries: []logrus.Entry{
				{
					Level:   logrus.InfoLevel,
					Message: "Status code metric updated - Code: 0, HTTP StatusCode: , Intel Error code: ",
					Data: logrus.Fields{
						SERVICE_STATUS_CODE_METRIC: INTEL_RS_REQUEST_FAILED,
						HTTP_STATUS_CODE_LABEL:     "",
						INTEL_ERROR_CODE_LABEL:     "",
					},
				},
			},
		},
		{
			msg: "SGX_RESET_NEEDED requires http code label ",
			metricUpdate: StatusCodeMetric{
				Status:         SGX_RESET_NEEDED,
				HttpStatusCode: "",
				IntelError:     "",
			},

			expectError: true,
			expectedLogEntries: []logrus.Entry{
				{
					Level:   logrus.InfoLevel,
					Message: "Status code metric updated - Code: 0, HTTP StatusCode: , Intel Error code: ",
					Data: logrus.Fields{
						SERVICE_STATUS_CODE_METRIC: SGX_RESET_NEEDED,
						HTTP_STATUS_CODE_LABEL:     "",
						INTEL_ERROR_CODE_LABEL:     "",
					},
				},
			},
		},
		{
			msg: "INVALID_REGISTRATION_REQUEST requires http code label ",
			metricUpdate: StatusCodeMetric{
				Status:         INVALID_REGISTRATION_REQUEST,
				HttpStatusCode: "",
				IntelError:     "",
			},

			expectError: true,
			expectedLogEntries: []logrus.Entry{
				{
					Level:   logrus.InfoLevel,
					Message: "Status code metric updated - Code: 0, HTTP StatusCode: , Intel Error code: ",
					Data: logrus.Fields{
						SERVICE_STATUS_CODE_METRIC: INVALID_REGISTRATION_REQUEST,
						HTTP_STATUS_CODE_LABEL:     "",
						INTEL_ERROR_CODE_LABEL:     "",
					},
				},
			},
		},
		{
			msg: "INTEL_RS_REQUEST_FAILED with http status code works fine",
			metricUpdate: StatusCodeMetric{

				Status:         INTEL_RS_REQUEST_FAILED,
				HttpStatusCode: "400",
				IntelError:     "",
			},
			expectError: false,
			expectedLogEntries: []logrus.Entry{
				{
					Level:   logrus.InfoLevel,
					Message: "Status code metric updated - Code: 12, HTTP StatusCode: 400, Intel Error code: ",
					Data: logrus.Fields{
						SERVICE_STATUS_CODE_METRIC: INTEL_RS_REQUEST_FAILED,
						HTTP_STATUS_CODE_LABEL:     "400",
						INTEL_ERROR_CODE_LABEL:     "",
					},
				},
			},
		},
	}
	for _, c := range cases {
		err := metricsRegistry.UpdateServiceStatusCodeMetric(c.metricUpdate)
		if c.expectError {
			assert.Error(t, err, c.msg)
		} else {
			for _, expectedEntry := range c.expectedLogEntries {
				thisLogEntryEqualTo(t, expectedEntry, *logHook.LastEntry(), c.msg)
			}
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
