package intelservices

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	mpmanagement "github.com/opensovereigncloud/cc-intel-platform-registration/internal/pkg/mp_management"
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

func (r *IntelService) RegisterPlatform(platformManifest mpmanagement.PlatformManifest) (metrics.StatusCodeMetric, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(constants.IntelRegirationRequestTimeoutInMinutes*time.Minute))
	defer cancel()
	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, constants.IntelPlatformRegistrationEndpoint, bytes.NewReader(platformManifest))
	if err != nil {
		r.log.Error("failed to create request", zap.Error(err))
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	// Execute request
	resp, err := client.Do(req)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			r.log.Error("request timeout to Intel registration service", zap.Error(err))
			return metrics.StatusCodeMetric{Status: metrics.IntelConnectFailed}, fmt.Errorf("connection timeout: %w", err)
		}
		r.log.Error("failed to send request to Intel registration service", zap.Error(err))
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
