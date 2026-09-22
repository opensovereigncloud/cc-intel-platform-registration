package mpmanagement

/*
#cgo CXXFLAGS: -std=c++17
#cgo LDFLAGS: -lmp_management -lstdc++

#include <stdlib.h>
#include "../../../third_party/mp_management/include/c_wrapper/mp_management.h"
#include "../../../third_party/mp_management/include/logger.h"

*/
import "C"

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sync"
)

var (
	globalZapLogger *zap.Logger
	loggerMutex     sync.RWMutex
)

func mapZapToCppLevel(level zapcore.Level) C.LogLevel {
	switch {
	case level <= zapcore.DebugLevel:
		return C.MP_REG_LOG_LEVEL_DEBUG
	case level == zapcore.InfoLevel:
		return C.MP_REG_LOG_LEVEL_INFO
	case level == zapcore.WarnLevel:
		return C.MP_REG_LOG_LEVEL_WARN
	case level >= zapcore.ErrorLevel:
		return C.MP_REG_LOG_LEVEL_ERROR
	default:
		return C.MP_REG_LOG_LEVEL_INFO
	}
}

func SetCgoZapLogger(l *zap.Logger) {
	if l == nil {
		return
	}

	loggerMutex.Lock()
	globalZapLogger = l
	loggerMutex.Unlock()
}

//export goZapLogBridge
func goZapLogBridge(component C.int, level C.int, msg *C.char) {
	loggerMutex.RLock()
	logger := globalZapLogger
	loggerMutex.RUnlock()

	if logger == nil {
		return
	}

	goMsg := C.GoString(msg)

	var compName string
	switch component {
	case 0:
		compName = "mp_uefi"
	case 1:
		compName = "mp_management"
	default:
		compName = "core"
	}

	compLogger := logger.With(zap.String("component", compName))

	switch level {
	case C.MP_REG_LOG_LEVEL_ERROR:
		compLogger.Error(goMsg)
	case C.MP_REG_LOG_LEVEL_WARN:
		compLogger.Warn(goMsg)
	case C.MP_REG_LOG_LEVEL_INFO:
		compLogger.Info(goMsg)
	case C.MP_REG_LOG_LEVEL_DEBUG:
		compLogger.Debug(goMsg)
	default:
		compLogger.Debug(goMsg)
	}
}

// MPManagement constants
const (
	MPResultCodeSuccess            = 0
	MPResultNoPendingData          = 1
	MPResultAlreadyRegistered      = 2
	MPResultMemoryError            = 3
	MPResultUefiInternalError      = 4
	MPResultUserInsufficientMemory = 5
	MPResultInvalidParameter       = 6
	MPResultSgxNotSupported        = 7
	MPResultUnexpectedError        = 8
	MPResultRedundantOperation     = 9
	MPResultNetworkError           = 10
	MPResultNotInitialized         = 11
	MPResultInsufficientPrivileges = 12
)

// MPManagement represents the Go wrapper for the mp_management functions
type MPManagement struct {
	initialized bool
}

type PlatformManifest []byte

// NewMPManagement creates a new instance of MPManagement
func NewMPManagement(efiVariablesPath string) *MPManagement {

	activeZapLevel := globalZapLogger.Level()
	glog_level := mapZapToCppLevel(activeZapLevel)

	C.mp_management_init(C.CString(efiVariablesPath), glog_level)
	return &MPManagement{initialized: true}
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
	var size C.uint32_t = C.UINT32_MAX
	buffer := make([]byte, size)

	operation_result := C.mp_management_get_platform_manifest((*C.uint8_t)(&buffer[0]), &size)

	if operation_result != MPResultCodeSuccess {
		return nil, fmt.Errorf("failed to get platform manifest uefi variable: %s", getErrorDescription(int(operation_result)))
	}

	return buffer[:size], nil
}

// IsMachineRegistered retrieves the machine registration status by reading the UEFI SgxRegistrationStatus.SgxRegistrationComplete variable flag
func (mp *MPManagement) IsMachineRegistered() (bool, error) {
	var status C.MpTaskStatus
	operation_result := C.mp_management_get_registration_status(&status)
	if operation_result != MPResultCodeSuccess {
		return false, fmt.Errorf("failed to get registration status uefi variable: %s", getErrorDescription(int(operation_result)))
	}
	return status == C.MP_TASK_COMPLETED, nil
}

func getErrorDescription(operation_result int) string {
	switch operation_result {
	case MPResultCodeSuccess:
		return "Code Success"
	case MPResultNoPendingData:
		return "No Pending Data"
	case MPResultAlreadyRegistered:
		return "Already Registered"
	case MPResultMemoryError:
		return "Memory Error"
	case MPResultUefiInternalError:
		return "Uefi Internal Error"
	case MPResultUserInsufficientMemory:
		return "User Insufficient Memory"
	case MPResultInvalidParameter:
		return "Invalid Parameter"
	case MPResultSgxNotSupported:
		return "Sgx Not Supported"
	case MPResultUnexpectedError:
		return "Unexpected Error"
	case MPResultRedundantOperation:
		return "Redundant Operation"
	case MPResultNetworkError:
		return "Network Error"
	case MPResultNotInitialized:
		return "NotInitialized"
	case MPResultInsufficientPrivileges:
		return "Insufficient Privileges"
	default:
		return "Unknown Error"
	}
}

// CompleteMachineRegistrationStatus sets the UEFI SgxRegistrationStatus.SgxRegistrationComplete flag to true
func (mp *MPManagement) CompleteMachineRegistrationStatus() error {
	operation_result := C.mp_management_set_registration_status_as_complete()
	if operation_result != MPResultCodeSuccess {
		return fmt.Errorf("failed to set the registration status uefi variable : %s", getErrorDescription(int(operation_result)))
	}
	return nil
}
