package sgxplatforminfo

/*
#cgo LDFLAGS: -lsgx_platform_info -lsgx_urts -lsgx_dcap_ql -lsgx_pce_logic -ldl -lpthread

#include <stdlib.h>
#include "../../../third_party/sgx_platform_info/src/sgx_platform_info.h"
*/
import "C"
import (
	"encoding/hex"
	"fmt"
	"unsafe"
)

const (
	SgxPlatformInfoSuccess = 61440

	SgxPlatformInfoUnexpectedError            = 61441
	SgxPlatformInfoInvalidParameterError      = 61442
	SgxPlatformInfoOutOfEPCError              = 61443
	SgxPlatformInfoInterfaceUnavailable       = 61444
	SgxPlatformInfoInvalidReportError         = 61445
	SgxPlatformInfoCryptoError                = 61446
	SgxPlatformInfoInvalidPrivilegeError      = 61447
	SgxPlatformInfoInvalidTCBError            = 61448
	SgxPlatformInfoEnclaveCreationFailedError = 61449
)

func getErrorDescription(operation_result int) string {
	switch operation_result {
	case SgxPlatformInfoUnexpectedError:
		return " Unexpected error"
	case SgxPlatformInfoInvalidParameterError:
		return "The parameter is incorrect"
	case SgxPlatformInfoOutOfEPCError:
		return "Not enough memory is available to complete this operation"
	case SgxPlatformInfoInterfaceUnavailable:
		return "SGX API is unavailable"
	case SgxPlatformInfoInvalidReportError:
		return "SGX report cannot be verified"
	case SgxPlatformInfoCryptoError:
		return " Cannot decrypt or verify ciphertext"
	case SgxPlatformInfoInvalidPrivilegeError:
		return "Not enough privilege to perform the operation"
	case SgxPlatformInfoInvalidTCBError:
		return "PCE could not sign at the requested TCB"
	case SgxPlatformInfoEnclaveCreationFailedError:
		return "The Enclave could not be created"
	default:
		return "Unknown Error"
	}
}

// SgxPlatformInfo contains the platform information retrieved from SGX
type SgxPlatformInfo struct {
	PCEInfo struct {
		PCEisvsvn string // PCE ISV Security Version Number (hex string)
		PCEID     string // PCE ID (hex string)
	}
	EncryptedPPID string // Encrypted Platform Provisioning ID (hex string)
	QeId          string // Quoting Enclave ID (hex string)
	CpuSvn        string // CPU Security Version Number (hex string)
}

// GetSgxPlatformInfo retrieves SGX platform information required to obtain a PCK certificate
// from Intel's Provisioning Certification Service (PCS) or PCCS.
//
// Returns:
//   - *SgxPlatformInfo: Platform information with all fields as hex-encoded strings
//   - error: nil on success, error with details on failure
func GetSgxPlatformInfo() (*SgxPlatformInfo, error) {
	var cPlatformInfo C.platform_info_t

	// Call C function to retrieve platform information
	result := C.get_platform_info(&cPlatformInfo)
	if result != SgxPlatformInfoSuccess {
		return nil, fmt.Errorf("failed to get platform info: error code %s", getErrorDescription(int(result)))
	}

	// Convert C struct to Go struct
	info := &SgxPlatformInfo{}

	// Extract PCE information
	// pce_id is uint16_t, formatted as 4-digit hex
	info.PCEInfo.PCEID = fmt.Sprintf("%04x", uint16(cPlatformInfo.pce_info.pce_id))
	// pce_isv_svn is uint16_t (sgx_isv_svn_t), formatted as 4-digit hex
	info.PCEInfo.PCEisvsvn = fmt.Sprintf("%04x", uint16(cPlatformInfo.pce_info.pce_isv_svn))

	// Extract Encrypted PPID (variable length, up to 384 bytes)
	// Uses actual size from encrypted_ppid_out_size field
	encrypted_ppid_raw := C.GoBytes(
		unsafe.Pointer(&cPlatformInfo.encrypted_ppid[0]),
		C.int(cPlatformInfo.encrypted_ppid_out_size),
	)
	info.EncryptedPPID = hex.EncodeToString(encrypted_ppid_raw)

	// Extract QE ID (fixed 16 bytes / 128 bits)
	qe_id_raw := C.GoBytes(
		unsafe.Pointer(&cPlatformInfo.qe_id[0]),
		C.QE_ID_SIZE,
	)
	info.QeId = hex.EncodeToString(qe_id_raw)

	// Extract CPU SVN (fixed 16 bytes / 128 bits)
	cpu_svn_raw := C.GoBytes(
		unsafe.Pointer(&cPlatformInfo.cpu_svn[0]),
		C.CPU_SVN_SIZE,
	)
	info.CpuSvn = hex.EncodeToString(cpu_svn_raw)

	return info, nil
}
