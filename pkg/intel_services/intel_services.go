package intelservices

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	mpmanagement "github.com/opensovereigncloud/cc-intel-platform-registration/internal/pkg/mp_management"
	sgxplatforminfo "github.com/opensovereigncloud/cc-intel-platform-registration/internal/pkg/sgx_platform_info"

	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/config"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/constants"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/metrics"
	"go.uber.org/zap"
)

// RegServiceEndpoints holds the list of registration and PCK retrieval URLs
type RegServiceEndpoints struct {
	registrationURL  string
	pckRetrievalURLs []string // PCCS URLs + Intel fallback
}

type IntelService struct {
	log        *zap.Logger
	httpClient *http.Client         // Reusable HTTP client with TLS config
	endpoints  *RegServiceEndpoints // URL configuration
}

// NewIntelService creates a new IntelService with configured HTTP client and endpoints
func NewIntelService(logger *zap.Logger, cfg *config.RegistrationServiceConfig) (*IntelService, error) {
	// Build TLS config (always uses system CA + optional custom CA for PCCS)
	tlsConfig, err := buildTLSConfig(cfg.PCCSCACertPath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %w", err)
	}

	// Create HTTP client with TLS config and connection pooling
	httpClient := &http.Client{
		Timeout: cfg.RequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			// Enable connection pooling for better performance
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     90 * constants.IntelRequestTimeout,
		},
	}

	// Build endpoint lists (PCCS URLs + Intel fallback)
	endpoints := buildEndpoints(cfg, logger)

	return &IntelService{
		log:        logger,
		httpClient: httpClient,
		endpoints:  endpoints,
	}, nil
}

func buildTLSConfig(caCertPath string, logger *zap.Logger) (*tls.Config, error) {
	// Base TLS config - always require TLS 1.2+
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Always start with system CA pool (needed for Intel API fallback)
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		logger.Warn("Failed to load system cert pool, using empty pool", zap.Error(err))
		caCertPool = x509.NewCertPool()
	}

	// If no custom CA cert path, use system CA bundle only
	if caCertPath == "" {
		logger.Debug("Using system CA bundle only (no custom CA configured)")
		return tlsConfig, nil
	}

	// Custom CA path provided - load certificates from directory
	fileInfo, err := os.Stat(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access CA certificate path %s: %w", caCertPath, err)
	}

	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("CA certificate path must be a directory, got file: %s", caCertPath)
	}

	certsLoaded := 0

	// Load all .crt files from directory
	logger.Debug("Loading custom CA certificates from directory",
		zap.String("path", caCertPath))

	entries, err := os.ReadDir(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate directory %s: %w", caCertPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only load .crt files
		if !strings.HasSuffix(entry.Name(), ".crt") {
			continue
		}

		certPath := filepath.Join(caCertPath, entry.Name())
		certData, err := os.ReadFile(certPath)
		if err != nil {
			logger.Warn("Failed to read CA certificate file, skipping",
				zap.String("file", certPath),
				zap.Error(err))
			continue
		}

		if caCertPool.AppendCertsFromPEM(certData) {
			certsLoaded++
			logger.Debug("Loaded custom CA certificate",
				zap.String("file", entry.Name()))
		} else {
			logger.Warn("Failed to parse CA certificate, skipping",
				zap.String("file", certPath))
		}
	}

	if certsLoaded == 0 {
		return nil, fmt.Errorf("no valid CA certificates found in directory %s", caCertPath)
	}

	// Successfully loaded custom CA(s) on top of system CA pool
	tlsConfig.RootCAs = caCertPool
	logger.Info("TLS configured with system CA pool + custom CA certificates",
		zap.Int("customCerts", certsLoaded))

	return tlsConfig, nil
}

// Platform registration: Always goes directly to Intel API
// PCK retrieval: Tries PCCS first (if configured), then Intel API as fallback
func buildEndpoints(cfg *config.RegistrationServiceConfig, logger *zap.Logger) *RegServiceEndpoints {
	endpoints := &RegServiceEndpoints{}

	// Platform registration always goes directly to Intel API
	endpoints.registrationURL = cfg.IntelRegistrationURL

	// PCK certificate retrieval: try PCCS first (if configured), then Intel as fallback
	if len(cfg.PCCSURLs) > 0 {
		logger.Info("Configuring PCCS endpoints for PCK retrieval",
			zap.Int("count", len(cfg.PCCSURLs)))
		for _, baseURL := range cfg.PCCSURLs {
			endpoints.pckRetrievalURLs = append(endpoints.pckRetrievalURLs,
				baseURL+"/sgx/certification/v4/pckcerts")
		}
	}

	// Always add Intel PCK retrieval URL as final fallback
	endpoints.pckRetrievalURLs = append(endpoints.pckRetrievalURLs,
		cfg.IntelPCKRetrievalURL)

	return endpoints
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

func (r *IntelService) RegisterPlatform(platformManifest mpmanagement.PlatformManifest, metricsRegistry *metrics.RegistrationServiceMetricsRegistry) (metrics.StatusCodeMetric, error) {
	// Platform registration only goes to Intel API (there should be exactly 1 URL)
	url := r.endpoints.registrationURL

	r.log.Debug("Attempting platform registration to Intel API",
		zap.String("url", url))

	metric, err := r.registerPlatformToEndpoint(url, platformManifest)

	if err == nil && metric.Status == metrics.PlatformRebootNeeded {
		r.log.Info("Platform registration successful",
			zap.String("url", url))
		return metric, nil
	}

	r.log.Error("Platform registration failed",
		zap.String("url", url),
		zap.Error(err))
	return metric, err
}

func (r *IntelService) registerPlatformToEndpoint(url string, platformManifest mpmanagement.PlatformManifest) (metrics.StatusCodeMetric, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(platformManifest))
	if err != nil {
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	// Execute request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return metrics.StatusCodeMetric{Status: metrics.IntelConnectFailed}, fmt.Errorf("connection timeout: %w", err)
		}
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return metrics.StatusCodeMetric{Status: metrics.PlatformRebootNeeded}, nil
	}
	errorCode := resp.Header.Get("Error-Code")
	return createIntelStatusCodeMetricForPlatformRegistration(resp.StatusCode, errorCode), nil
}

// RetrievePCK attempts to retrieve PCK certificate
// It tries each endpoint in order (PCCS first, then Intel) until one succeeds
func (r *IntelService) RetrievePCK(platformInfo *sgxplatforminfo.SgxPcePlatformInfo, metricsRegistry *metrics.RegistrationServiceMetricsRegistry) (metrics.StatusCodeMetric, error) {
	var lastErr error
	var lastMetric metrics.StatusCodeMetric

	// Try each PCK retrieval endpoint in order
	for i, baseURL := range r.endpoints.pckRetrievalURLs {
		isPCCS := i < len(r.endpoints.pckRetrievalURLs)-1
		endpointType := "intel"
		if isPCCS {
			endpointType = "pccs"
		}

		requestURL := fmt.Sprintf("%s?encrypted_ppid=%s&pceid=%s",
			baseURL, platformInfo.EncryptedPPID, platformInfo.PCEInfo.PCEID)

		r.log.Debug("Attempting PCK retrieval",
			zap.String("url", requestURL),
			zap.String("endpointType", endpointType),
			zap.Int("attemptNumber", i+1))

		metric, err := r.retrievePCKFromEndpoint(requestURL)

		// Success - return immediately
		if err == nil && metric.Status == metrics.PlatformDirectlyRegistered {
			r.log.Info("PCK retrieval successful",
				zap.String("url", requestURL),
				zap.String("endpointType", endpointType),
				zap.Int("attemptNumber", i+1))

			return metric, nil
		}

		// Store error and continue
		lastErr = err
		lastMetric = metric

		r.log.Warn("PCK retrieval failed, trying next endpoint",
			zap.String("url", requestURL),
			zap.String("endpointType", endpointType),
			zap.Error(err))
	}

	// All endpoints failed
	r.log.Error("PCK retrieval failed on all endpoints",
		zap.Error(lastErr))
	return lastMetric, lastErr
}

// retrievePCKFromEndpoint attempts PCK retrieval from a single endpoint
func (r *IntelService) retrievePCKFromEndpoint(requestURL string) (metrics.StatusCodeMetric, error) {
	req, err := http.NewRequest(http.MethodGet, requestURL, http.NoBody)
	if err != nil {
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("connection timeout: %w", err)
		}
		return metrics.CreateUnknownErrorStatusCodeMetric(), fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return metrics.StatusCodeMetric{Status: metrics.PlatformDirectlyRegistered}, nil
	}
	errorCode := resp.Header.Get("Error-Code")
	return createIntelStatusCodeMetricForDirectRegistration(resp.StatusCode, errorCode), nil
}
