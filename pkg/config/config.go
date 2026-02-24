package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/constants"
)

// RegistrationServiceConfig holds all configuration for the registration service
type RegistrationServiceConfig struct {
	// PCCS configuration
	PCCSURLs       []string // Parsed from CC_PCCS_URLS
	PCCSCACertPath string   // From CC_PCCS_CA_CERT_PATH (directory with custom CA certificates)

	// Intel fallback endpoints
	IntelRegistrationURL string
	IntelPCKRetrievalURL string

	// HTTP client settings
	RequestTimeout time.Duration

	// Service settings
	RegistrationInterval time.Duration
	ServicePort          int
}

// LoadRegistrationServiceConfig loads configuration from environment variables
func LoadRegistrationServiceConfig() (*RegistrationServiceConfig, error) {
	config := &RegistrationServiceConfig{
		IntelRegistrationURL: constants.IntelPlatformRegistrationEndpoint,
		IntelPCKRetrievalURL: constants.IntelPckRetrievalEndpoint,
		RequestTimeout:       constants.IntelRequestTimeout,
	}

	// Parse PCCS URLs (optional)
	pccsURLsEnv := os.Getenv(constants.PCCSURLsEnv)
	if pccsURLsEnv != "" {
		urls := strings.Split(pccsURLsEnv, ",")
		for _, rawURL := range urls {
			rawURL = strings.TrimSpace(rawURL)
			if rawURL == "" {
				continue
			}

			// Validate URL format and require HTTPS
			parsedURL, err := url.Parse(rawURL)
			if err != nil {
				return nil, fmt.Errorf("invalid PCCS URL '%s': %w", rawURL, err)
			}
			if parsedURL.Scheme != "https" {
				return nil, fmt.Errorf("PCCS URL must use HTTPS: '%s'", rawURL)
			}

			// Remove trailing slash for consistency
			rawURL = strings.TrimSuffix(rawURL, "/")
			config.PCCSURLs = append(config.PCCSURLs, rawURL)
		}
	}

	// Load CA cert path (optional - directory containing custom CA certificates for PCCS)
	config.PCCSCACertPath = os.Getenv(constants.PCCSCACertPathEnv)

	// Load registration interval
	intervalMinutes := constants.DefaultRegistrationServiceIntervalInMinutes
	if intervalEnv := os.Getenv(constants.DefaultRegistrationServiceIntervalInMinutesEnv); intervalEnv != "" {
		parsed, err := strconv.Atoi(intervalEnv)
		if err != nil {
			return nil, fmt.Errorf("invalid registration interval: %w", err)
		}
		intervalMinutes = parsed
	}
	config.RegistrationInterval = time.Duration(intervalMinutes) * time.Minute

	// Load service port
	servicePort := constants.DefaultRegistrationServicePort
	if portEnv := os.Getenv(constants.RegistrationServicePortEnv); portEnv != "" {
		parsed, err := strconv.Atoi(portEnv)
		if err != nil {
			return nil, fmt.Errorf("invalid service port: %w", err)
		}
		servicePort = parsed
	}
	config.ServicePort = servicePort

	return config, nil
}
