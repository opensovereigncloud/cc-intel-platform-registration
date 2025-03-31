package intelservices

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	mpmanagement "github.com/opensovereigncloud/cc-intel-platform-registration/internal/pkg/mp_management"
	sgxplatforminfo "github.com/opensovereigncloud/cc-intel-platform-registration/internal/pkg/sgx_platform_info"

	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/constants"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/metrics"
	"go.uber.org/zap"
)

type IntelService struct {
	log *zap.Logger
}

func NewIntelService(logger *zap.Logger) *IntelService {
	return &IntelService{
		log: logger,
	}
}

func createIntelStatusCodeMetricForPlatformRegistration(httpStatusCode int, intelErrorCode string) metrics.StatusCodeMetric {
	var Status metrics.StatusCode
	if httpStatusCode >= http.StatusBadRequest && httpStatusCode < http.StatusInternalServerError {
		Status = metrics.InvalidRegistrationRequest
	} else {
		Status = metrics.IntelRegServiceRequestFailed
	}
	return metrics.StatusCodeMetric{
		Status:         Status,
		HttpStatusCode: strconv.Itoa(httpStatusCode),
		IntelError:     intelErrorCode,
	}
}

func createIntelStatusCodeMetricForDirectRegistration(httpStatusCode int, intelErrorCode string) metrics.StatusCodeMetric {

	var Status metrics.StatusCode
	if httpStatusCode == http.StatusNotFound {
		Status = metrics.SgxResetNeeded
	} else {
		Status = metrics.RetryNeeded
	}

	return metrics.StatusCodeMetric{
		Status:         Status,
		HttpStatusCode: strconv.Itoa(httpStatusCode),
		IntelError:     intelErrorCode,
	}
}

func (r *IntelService) RegisterPlatform(platformManifest mpmanagement.PlatformManifest) (metrics.StatusCodeMetric, error) {
	client := &http.Client{
		Timeout: constants.IntelRequestTimeout,
	}

	req, err := http.NewRequest(http.MethodPost, constants.IntelPlatformRegistrationEndpoint, bytes.NewReader(platformManifest))
	if err != nil {
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	// Execute request
	resp, err := client.Do(req)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return metrics.StatusCodeMetric{Status: metrics.IntelConnectFailed}, fmt.Errorf("connection timeout: %w", err)
		}
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return metrics.StatusCodeMetric{Status: metrics.PlatformRebootNeeded}, nil
	} else {
		errorCode := resp.Header.Get("Error-Code")
		return createIntelStatusCodeMetricForPlatformRegistration(resp.StatusCode, errorCode), nil
	}

}

func (r *IntelService) RetrievePCK(platformInfo *sgxplatforminfo.SgxPcePlatformInfo) (metrics.StatusCodeMetric, error) {

	client := &http.Client{
		Timeout: constants.IntelRequestTimeout,
	}

	requestURL := fmt.Sprintf("%s?encrypted_ppid=%s&pceid=%s",
		constants.IntelPckRetrievalEndpoint, platformInfo.EncryptedPPID, platformInfo.PCEInfo.PCEID)
	req, err := http.NewRequest(http.MethodGet, requestURL, http.NoBody)

	if err != nil {
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := client.Do(req)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("connection timeout: %w", err)
		}
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return metrics.StatusCodeMetric{Status: metrics.PlatformDirectlyRegistered}, nil
	} else {
		errorCode := resp.Header.Get("Error-Code")
		return createIntelStatusCodeMetricForDirectRegistration(resp.StatusCode, errorCode), nil
	}

}
