package sgxplatforminfo

/*
#cgo LDFLAGS: -L../../../build/lib -Wl,-rpath,$ORIGIN/build/lib -lsgxplatforminfo -lsgx_urts -lsgx_dcap_ql -lsgx_pce_logic  -ldl -lpthread

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

// GetSgxPcePlatformInfo gets the PCE information using SGX
func GetSgxPcePlatformInfo() (*SgxPcePlatformInfo, error) {
	var cPlatformInfo C.platform_info_t

	result := C.get_platform_info(&cPlatformInfo)
	if result != C.SGX_PCE_SUCCESS {
		return nil, fmt.Errorf("failed to get platform info: error code %d", result)
	}

	// Convert C struct to Go struct
	info := &SgxPcePlatformInfo{}
	info.PCEInfo.PCEID = fmt.Sprintf("%04x", uint16(cPlatformInfo.pce_info.pce_id))
	info.PCEInfo.PCEisvsvn = fmt.Sprintf("0x%02x", uint16(cPlatformInfo.pce_info.pce_isv_svn))

	encryptted_ppid_raw := C.GoBytes(unsafe.Pointer(&cPlatformInfo.encrypted_ppid[0]), C.int(cPlatformInfo.encrypted_ppid_out_size))

	info.EncryptedPPID = hex.EncodeToString(encryptted_ppid_raw)

	return info, nil
}
