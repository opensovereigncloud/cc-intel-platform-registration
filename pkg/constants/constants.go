package constants

import "time"

const DefaultRegistrationServiceIntervalInMinutes = 60
const DefaultRegistrationServiceIntervalInMinutesEnv = "CC_IPR_REGISTRATION_INTERVAL_MINUTES"

const DefaultRegistrationServicePort = 8080
const RegistrationServicePortEnv = "CC_IPR_REGISTRATION_SERVICE_PORT"

// PCCS configuration
const PCCSURLsEnv = "CC_PCCS_URLS"
const PCCSCACertPathEnv = "CC_PCCS_CA_CERT_PATH" // Directory path for custom CA certificates

// Intel endpoint constants (used as fallback)
const IntelPlatformRegistrationEndpoint = "https://api.trustedservices.intel.com/sgx/registration/v1/platform"
const IntelPckRetrievalEndpoint = "https://api.trustedservices.intel.com/sgx/certification/v4/pckcert"
const IntelRequestTimeout = 2 * time.Minute
