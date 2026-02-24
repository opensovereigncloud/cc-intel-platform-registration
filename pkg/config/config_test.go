package config

import (
	"os"
	"testing"

	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/constants"
)

func TestLoadRegistrationServiceConfig_NoCACertPath(t *testing.T) {
	tests := []struct {
		name        string
		pccsURLs    string
		caCertPath  string
		expectError bool
	}{
		{
			name:        "No PCCS URLs - no custom CA needed",
			pccsURLs:    "",
			caCertPath:  "",
			expectError: false,
		},
		{
			name:        "PCCS URLs without custom CA - valid (uses system CA)",
			pccsURLs:    "https://pccs.example.com",
			caCertPath:  "",
			expectError: false,
		},
		{
			name:        "PCCS URLs with custom CA directory - valid",
			pccsURLs:    "https://pccs.example.com",
			caCertPath:  "/etc/ssl/pccs-certs",
			expectError: false,
		},
		{
			name:        "Multiple PCCS URLs without custom CA - valid (uses system CA)",
			pccsURLs:    "https://pccs1.example.com,https://pccs2.example.com",
			caCertPath:  "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			os.Clearenv()
			if tt.pccsURLs != "" {
				os.Setenv(constants.PCCSURLsEnv, tt.pccsURLs)
			}
			if tt.caCertPath != "" {
				os.Setenv(constants.PCCSCACertPathEnv, tt.caCertPath)
			}

			// Call the function
			cfg, err := LoadRegistrationServiceConfig()

			// Check results
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if cfg == nil {
					t.Errorf("Expected config to be non-nil")
				}
			}
		})
	}
}
