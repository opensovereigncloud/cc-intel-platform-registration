/*
 * Copyright(c) 2011-2026 Intel Corporation
 *
 * SPDX-License-Identifier: BSD-3-Clause
 */

#include <climits>
#include <cstddef>
#include <cstring>
#include <cassert>
#include <memory>
#include <string.h>
#include <type_traits>

#include "FSUefi.h"
#include <regex.h>
#include "MPUefi.h"
#include "UefiVar.h"
#include "ByteSerialization.h"
#include "logger.h"

#define REGISTRATION_COMPLETE_BIT_MASK 0x0001
#define PACKAGE_INFO_COMPLETE_BIT_MASK 0x0002

// defines for values verification
#define MP_VERIFY_UEFI_STRUCT_READ 1
#define MP_VERIFY_UEFI_VERSION_READ 1

namespace
{

  using UefiVersion = decltype(std::declval<SgxUefiVar>().version);
  using UefiSize = decltype(std::declval<SgxUefiVar>().size);
  using S3mUefiVersion = decltype(std::declval<S3mUefiVar>().version);
  using S3mUefiSize = decltype(std::declval<S3mUefiVar>().size);

  using mp::parseBytesLE;
  using mp::writeBytesLE;

  struct RequestInfo
  {
    UefiVersion version;
    S3mUefiSize uefiVarSize;
    size_t requiredSize, headerOffset;
  };

  RequestInfo getRequestInfo(const uint8_t *request)
  {
    static_assert(sizeof(UefiVersion) == sizeof(S3mUefiVersion), "Version variables sizes needs to equal");
    static_assert(sizeof(S3mUefiSize) >= sizeof(UefiSize), "S3mSize needs to able to hold legacy size");

    const UefiVersion version = parseBytesLE<UefiVersion>(request);
    if (version == MP_BIOS_UEFI_VARIABLE_VERSION_3)
    {
      // PLATFORM_MANIFEST_V2
      const S3mUefiSize varSize = parseBytesLE<S3mUefiSize>(request, sizeof(UefiVersion));
      const size_t required = sizeof(UefiVersion) + sizeof(S3mUefiSize) + varSize;
      S3mUefiVar var{};
      return {
          version,
          varSize,
          required,
          size_t(std::addressof(var.header)) - size_t(std::addressof(var))};
    }

    // PLATFORM_MANIFEST V1
    const UefiSize varSize = parseBytesLE<UefiSize>(request, sizeof(UefiVersion));
    const size_t required = sizeof(UefiVersion) + sizeof(UefiSize) + varSize;
    SgxUefiVar var{};
    return {
        version,
        static_cast<S3mUefiSize>(varSize),
        required,
        size_t(std::addressof(var.header)) - size_t(std::addressof(var))};
  };

} // namespace

std::unique_ptr<FSUefi> initUefi(std::string path, LogLevel logLevel)
{
  if (path.empty())
  {
    return std::make_unique<FSUefi>(EFIVARS_FILE_SYSTEM, logLevel);
  }
  return std::make_unique<FSUefi>(std::move(path), logLevel);
}

MPUefi::MPUefi(std::unique_ptr<FSUefi> uefi)
    : m_uefi{std::move(uefi)}, m_logLevel{m_uefi->getLogLevel()}
{
}

MpResult MPUefi::getRequestType(MpRequestType &type)
{
  /*
   * We need to check GUID in uefi variable.
   * We expect either PlatformManifest_GUID or AddRequest_GUID.
   *
   * GUID is first entry in header field, which itself is a third field in
   * UEFI structure.
   *
   * First entry is version, which is uint16_t and second field is size,
   * which is either uint16_t or uint32_t depending on platform manifest version.
   *
   * So first, we check uint16_t version, and then move data pointer by appropriate value
   * depending on manifest version.
   *
   */

  size_t varDataSize = 0;
  std::unique_ptr<uint8_t[]> requestUefi{m_uefi->readUEFIVar(UEFI_VAR_SERVER_REQUEST, varDataSize)};
  if (!requestUefi)
  {
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequestType: SgxRegistrationServerRequest UEFI variable was not found.\n");

    type = MP_REQ_NONE;
    return MP_SUCCESS;
  }

  // Ensure the buffer is large enough for getRequestInfo() to safely parse
  // both the version field (uint16_t) and the widest size field (uint32_t for
  // V3).  All three fields are at fixed offsets 0-5 in the raw UEFI variable.
  if (varDataSize < sizeof(UefiVersion) + sizeof(S3mUefiSize))
  {
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequestType: SgxRegistrationServerRequest UEFI variable is too small to parse header.\n");
    type = MP_REQ_NONE;
    return MP_UEFI_INTERNAL_ERROR;
  }

  const auto [version, uefiVarSize, requiredSize, headerOffset] = getRequestInfo(requestUefi.get());

#ifdef MP_VERIFY_UEFI_VERSION_READ
  // structure version check
  if (!(version == MP_BIOS_UEFI_VARIABLE_VERSION_1 || version == MP_BIOS_UEFI_VARIABLE_VERSION_2 || version == MP_BIOS_UEFI_VARIABLE_VERSION_3))
  {
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequestType: version check failed. version: %d\n", version);
    return MP_UEFI_INTERNAL_ERROR;
  }
#endif

#ifdef MP_VERIFY_UEFI_STRUCT_READ
  if (varDataSize != requiredSize)
  {
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequestType: SgxRegistrationServerRequest UEFI size is invalid.\n");
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequestType: actual size: %zu, expected size: %zu\n", varDataSize, requiredSize);

    return MP_UEFI_INTERNAL_ERROR;
  }
#endif
  // Ensure the buffer is large enough to contain a full GUID at headerOffset.
  // varSize is BIOS-controlled and may be smaller than GUID_SIZE even when
  // varDataSize == requiredSize, which would cause an over-read below.
  if (varDataSize < headerOffset + GUID_SIZE)
  {
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequestType: SgxRegistrationServerRequest UEFI variable too small to contain GUID.\n");
    type = MP_REQ_NONE;
    return MP_UEFI_INTERNAL_ERROR;
  }

  const uint8_t *guidPtr = requestUefi.get() + headerOffset;

  // TODO:
  // Do we need additional checks here?
  // For MP_BIOS_UEFI_VARIABLE_VERSION_1 there can be only manifest
  // For MP_BIOS_UEFI_VARIABLE_VERSION_2 there can be manifest or add package
  // For MP_BIOS_UEFI_VARIABLE_VERSION_3 there can be manifest V2

  // set request type
  if (0 == memcmp(guidPtr, PlatformManifest_GUID, GUID_SIZE))
  {
    type = MP_REQ_REGISTRATION;
  }
  else if (0 == memcmp(guidPtr, AddRequest_GUID, GUID_SIZE))
  {
    type = MP_REQ_ADD_PACKAGE;
  }
  else
  {
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequestType: request GUID doesn't match expected GUID's\n");
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequestType: GUID from SgxRegistrationServerRequest UEFI:\n");
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX%02hhX\n",
                     guidPtr[0], guidPtr[1], guidPtr[2], guidPtr[3], guidPtr[4], guidPtr[5],
                     guidPtr[6], guidPtr[7], guidPtr[8], guidPtr[9], guidPtr[10], guidPtr[11],
                     guidPtr[12], guidPtr[13], guidPtr[14], guidPtr[15]);

    return MP_UEFI_INTERNAL_ERROR;
  }

  return MP_SUCCESS;
}

MpResult MPUefi::getRequest(uint8_t *request, uint32_t &requestSize)
{
  /*
   *
   * 1. Read Version which is uint16_t at the begining.
   * 2. If Version == 1 or 2 then it's MANIFEST V1. If not goto 3.
   *  2a. Next Size is uint16_t on next bytes
   *  2b. Ensure Size is not bigger than passed request size in argument
   *  2c. Return Size count bytes to request pointer
   * 3. If UEFI variable werson == 3 then it's MANIFEST V2
   *  3a. Next Size is uint32_t on next bytes
   *  3b. Ensure Size is not bigger than passed request size in argument
   *  3c. Return Size count bytes to request pointer
   *
   */

  size_t varDataSize{0};
  std::unique_ptr<uint8_t[]> requestUefi{m_uefi->readUEFIVar(UEFI_VAR_SERVER_REQUEST, varDataSize)};
  if (!requestUefi)
    return MP_NO_PENDING_DATA;

  // Ensure the buffer is large enough for getRequestInfo() to safely parse
  // both the version field (uint16_t) and the widest size field (uint32_t for
  // V3).  All three fields are at fixed offsets 0-5 in the raw UEFI variable.
  if (varDataSize < sizeof(UefiVersion) + sizeof(S3mUefiSize))
  {
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequest: SgxRegistrationServerRequest UEFI variable is too small to parse header.\n");
    return MP_UEFI_INTERNAL_ERROR;
  }

  const auto [version, uefiVarSize, requiredSize, headerOffset] = getRequestInfo(requestUefi.get());

#ifdef MP_VERIFY_UEFI_VERSION_READ
  if (!(version == MP_BIOS_UEFI_VARIABLE_VERSION_1 || version == MP_BIOS_UEFI_VARIABLE_VERSION_2 || version == MP_BIOS_UEFI_VARIABLE_VERSION_3))
  {
    uefi_log_message(
        MP_REG_LOG_LEVEL_ERROR,
        "getRequest: version check failed, UEFI version: %d, expected UEFI version: %d or %d or %d\n",
        version, MP_BIOS_UEFI_VARIABLE_VERSION_1,
        MP_BIOS_UEFI_VARIABLE_VERSION_2, MP_BIOS_UEFI_VARIABLE_VERSION_3);

    return MP_INVALID_PARAMETER;
  }
#endif

#ifdef MP_VERIFY_UEFI_STRUCT_READ
  if (varDataSize != requiredSize)
  {
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequest: SgxRegistrationServerRequest UEFI size is invalid.\n");
    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequest: actual size: %zu, expected size: %zu\n", varDataSize, requiredSize);

    return MP_UEFI_INTERNAL_ERROR;
  }
#endif

  if (request)
  {
    if (requestSize < uefiVarSize)
    {
      uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequest: Request buffer too small for pending request, given size: %d, actual size: %d.\n", requestSize, uefiVarSize);
      return MP_USER_INSUFFICIENT_MEM;
    }

    // Unconditional bound check: headerOffset + uefiVarSize must not exceed
    // the allocation.  Use an overflow-safe form to guard against a crafted
    // uefiVarSize that would wrap on 32-bit builds.
    // The MP_VERIFY_UEFI_STRUCT_READ guard above already enforces this when
    // enabled, but add a hard stop for defense-in-depth.
    if (headerOffset > varDataSize || static_cast<size_t>(uefiVarSize) > varDataSize - headerOffset)
    {
      uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRequest: UEFI variable payload overruns allocated buffer.\n");
      return MP_UEFI_INTERNAL_ERROR;
    }

    requestSize = uefiVarSize;
    std::memcpy(request, requestUefi.get() + headerOffset, uefiVarSize);
  }

  return MP_SUCCESS;
}

MpResult MPUefi::getRegistrationStatus(MpRegistrationStatus &status)
{
  MpResult res = MP_SUCCESS;
  size_t varDataSize = 0;
  RegistrationStatusUEFI *statusUefi = 0;

  do
  {
    statusUefi = (RegistrationStatusUEFI *)m_uefi->readUEFIVar(UEFI_VAR_STATUS, varDataSize);
    if (statusUefi == 0 || varDataSize != sizeof(RegistrationStatusUEFI))
    {
      uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRegistrationStatus: SgxRegistrationStatus UEFI variable was not found or size not as expected.\n");
      uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRegistrationStatus: SgxRegistrationStatus actual size: %zu, expected size: %zu\n", varDataSize, sizeof(RegistrationStatusUEFI));
      res = MP_UEFI_INTERNAL_ERROR;
      break;
    }

    // structure version check
    if (statusUefi->version != MP_BIOS_UEFI_VARIABLE_VERSION_1)
    {
      uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRegistrationStatus: version check failed.\n");
      res = MP_UEFI_INTERNAL_ERROR;
      break;
    }

    // uefi structure size check
    if (statusUefi->size != sizeof(statusUefi->status) + sizeof(statusUefi->errorCode))
    {
      uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRegistrationStatus: SgxRegistrationStatus structure size not as expected.\n");
      uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "getRegistrationStatus: statusUefi->size: %d, sizeof(statusUefi->status): %zu, sizeof(statusUefi->errorCode): %zu\n",
                       statusUefi->size, sizeof(statusUefi->status), sizeof(statusUefi->errorCode));
      res = MP_UEFI_INTERNAL_ERROR;
      break;
    }
    memset(&status, 0, sizeof(status));

    if (statusUefi->status & REGISTRATION_COMPLETE_BIT_MASK)
    {
      status.registrationStatus = MP_TASK_COMPLETED;
    }
    if (statusUefi->status & PACKAGE_INFO_COMPLETE_BIT_MASK)
    {
      status.packageInfoStatus = MP_TASK_COMPLETED;
    }
    status.errorCode = (RegistrationErrorCode)statusUefi->errorCode;
  } while (0);

  if (statusUefi)
  {
    delete[] (uint8_t *)statusUefi;
  }

  return res;
}

MpResult MPUefi::setRegistrationStatus(const MpRegistrationStatus &status)
{
  MpResult res = MP_SUCCESS;
  RegistrationStatusUEFI statusUefi;

  // zero all response uefi structure
  memset(&(statusUefi), 0, sizeof(statusUefi));

  do
  {
    statusUefi.version = MP_BIOS_UEFI_VARIABLE_VERSION_1;
    statusUefi.size = (uint16_t)(sizeof(statusUefi.status) + sizeof(statusUefi.errorCode));

    if (status.registrationStatus == MP_TASK_COMPLETED)
    {
      statusUefi.status |= REGISTRATION_COMPLETE_BIT_MASK;
    }
    if (status.packageInfoStatus == MP_TASK_COMPLETED)
    {
      statusUefi.status |= PACKAGE_INFO_COMPLETE_BIT_MASK;
    }
    uefi_log_message(MP_REG_LOG_LEVEL_DEBUG, "setRegistrationStatus: status.status = 0x%02x, statusUefi.status = 0x%02x.\n", status.status, statusUefi.status);

    statusUefi.errorCode = status.errorCode;

    // write registration status to uefi
    int numOfBytes = m_uefi->writeUEFIVar(UEFI_VAR_STATUS, (const uint8_t *)(&statusUefi), sizeof(statusUefi), false);
    if (numOfBytes != sizeof(statusUefi))
    {
      if (numOfBytes == -1)
      {
        uefi_log_message(MP_REG_LOG_LEVEL_WARN, "fail to write registration status uefi variable, maybe it is in read-only mode.\n");
        res = MP_INSUFFICIENT_PRIVILEGES;
        break;
      }
      uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "setRegistrationStatus: failed to write uefi variable.\n");
      res = MP_UEFI_INTERNAL_ERROR;
      break;
    }
  } while (0);

  return res;
}
