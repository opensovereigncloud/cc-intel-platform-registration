package sgxplatforminfo

/*
#cgo LDFLAGS: -lsgx_platform_info -lsgx_urts -lsgx_dcap_ql -lsgx_pce_logic  -ldl -lpthread

#include <stdlib.h>
#include "../../../third_party/sgx_platform_info/src/sgx_platform_info.h"
*/
import "C"
import (
	"encoding/hex"
	"fmt"
	"unsafe"
)

// SgxPcePlatformInfo contains the PCE information gotten from the PCE enclave
type SgxPcePlatformInfo struct {
	PCEInfo struct {
		PCEisvsvn string
		PCEID     string
	}
	EncryptedPPID string //

}

const (
	SgxPcePlatformSuccess = 61440
	
	SgxPcePlatformUnexpectedError            = 61441
	SgxPcePlatformInvalidParameterError      = 61442
	SgxPcePlatformOutOfEPCError              = 61443
	SgxPcePlatformInterfaceUnavailable       = 61444
	SgxPcePlatformInvalidReportError         = 61445
	SgxPcePlatformCryptoError                = 61446
	SgxPcePlatformInvalidPrivilegeError      = 61447
	SgxPcePlatformInvalidTCBError            = 61448
	SgxPcePlatformEnclaveCreationFailedError = 61449
)

func getErrorDescription(operation_result int) string {
	switch operation_result {
	case SgxPcePlatformUnexpectedError:
		return " Unexpected error"
	case SgxPcePlatformInvalidParameterError:
		return "The parameter is incorrect"
	case SgxPcePlatformOutOfEPCError:
		return "Not enough memory is available to complete this operation"
	case SgxPcePlatformInterfaceUnavailable:
		return "SGX API is unavailable"
	case SgxPcePlatformInvalidReportError:
		return "SGX report cannot be verified"
	case SgxPcePlatformCryptoError:
		return " Cannot decrypt or verify ciphertext"
	case SgxPcePlatformInvalidPrivilegeError:
		return "Not enough privilege to perform the operation"
	case SgxPcePlatformInvalidTCBError:
		return "PCE could not sign at the requested TCB"
	case SgxPcePlatformEnclaveCreationFailedError:
		return "The Enclave could not be created"
	default:
		return "Unknown Error"
	}
}

// GetSgxPcePlatformInfo gets the PCE information using SGX
func GetSgxPcePlatformInfo() (*SgxPcePlatformInfo, error) {
	var cPlatformInfo C.platform_info_t

	result := C.get_platform_info(&cPlatformInfo)
	if result != SgxPcePlatformSuccess {
		return nil, fmt.Errorf("failed to get the sgx pce platform info: error code %s", getErrorDescription(int(result)))
	}

	// Convert C struct to Go struct
	info := &SgxPcePlatformInfo{}
	info.PCEInfo.PCEID = fmt.Sprintf("%04x", uint16(cPlatformInfo.pce_info.pce_id))
	info.PCEInfo.PCEisvsvn = fmt.Sprintf("0x%02x", uint16(cPlatformInfo.pce_info.pce_isv_svn))

	encryptted_ppid_raw := C.GoBytes(unsafe.Pointer(&cPlatformInfo.encrypted_ppid[0]), C.int(cPlatformInfo.encrypted_ppid_out_size))

	info.EncryptedPPID = hex.EncodeToString(encryptted_ppid_raw)

	return info, nil
}
