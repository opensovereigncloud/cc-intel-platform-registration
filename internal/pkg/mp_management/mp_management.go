package mp_management

/*
#cgo CXXFLAGS: -std=c++17
#cgo LDFLAGS: -L../../../build/lib -Wl,-rpath,$ORIGIN/build/lib -lmp_management -lstdc++

#include <stdlib.h>
#include "../../../third_party/mp_management/src/include/c_wrapper/mp_management.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

const (
	MaxPathSize     = 256
	MaxResponseSize = 1024 * 30
	MaxRequestSize  = 1024 * 56
	MaxDataSize     = 1024 * 30
)

// operation_result codes
const (
	MPSuccess             = 0
	MPNoPendingData       = 1
	MPAlreadyRegistered   = 2
	MPMemError            = 3
	MPUefiInternalError   = 4
	MPUserInsufficientMem = 5
	MPInvalidParameter    = 6
	MPSgxNotSupported     = 7
	MPUnexpectedError     = 8
	MPRedundantOperation  = 9
	MPNetworkError        = 10
	MPNotInitialized      = 11
)

// TaskStatus constants
const (
	MPTaskInProgress = 0
	MPTaskCompleted  = 1
)

// MPManagement represents the Go wrapper for the mp_management functions
type MPManagement struct {
	initialized bool
}

type PlatformManifest []byte

// NewMPManagement creates a new instance of MPManagement
func NewMPManagement() (*MPManagement, error) {
	cPath := C.CString("")
	defer C.free(unsafe.Pointer(cPath))

	C.mp_management_init(cPath)

	return &MPManagement{initialized: true}, nil
}

// Close terminates the MPManagement instance
func (mp *MPManagement) Close() {
	if mp.initialized {
		C.mp_management_terminate()
		mp.initialized = false
	}
}

// GetPlatformManifest retrieves the platform manifest by reading the UEFI SgxRegistrationServerRequest
func (mp *MPManagement) GetPlatformManifest() (PlatformManifest, error) {
	var size C.uint16_t = MaxRequestSize
	buffer := make([]byte, size)

	operation_result := C.mp_management_get_platform_manifest((*C.uint8_t)(&buffer[0]), &size)

	if operation_result != MPSuccess {
		return nil, fmt.Errorf("failed to get platform manifest: %d", operation_result)
	}

	return buffer[:size], nil
}

// IsMachineRegistered retrieves the machine registration status by reading the UEFI SgxRegistrationStatus.SgxRegistrationComplete variable flag
func (mp *MPManagement) IsMachineRegistered() (bool, error) {
	var status C.MpMachineRegistrationStatus
	operation_result := C.mp_management_get_registration_status(&status)
	if operation_result != MPSuccess {
		return false, fmt.Errorf("failed to get registration status: %d", operation_result)
	}
	return status == C.MP_MACHINE_REGISTERED, nil
}

// CompleteMachineRegistrationStatus sets the UEFI SgxRegistrationStatus.SgxRegistrationComplete flag to true
func (mp *MPManagement) CompleteMachineRegistrationStatus() error {
	operation_result := C.mp_management_set_registration_status_as_complete()
	if operation_result != MPSuccess {
		return fmt.Errorf("failed to get registration status: %d", operation_result)
	}
	return nil
}
