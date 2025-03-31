/*
 * Copyright (C) 2011-2021 Intel Corporation. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 *
 *   * Redistributions of source code must retain the above copyright
 *     notice, this list of conditions and the following disclaimer.
 *   * Redistributions in binary form must reproduce the above copyright
 *     notice, this list of conditions and the following disclaimer in
 *     the documentation and/or other materials provided with the
 *     distribution.
 *   * Neither the name of Intel Corporation nor the names of its
 *     contributors may be used to endorse or promote products derived
 *     from this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */
/**
 * File: MPUefi.cpp
 *
 * Description: Linux specific implementation for the MPUefi class to
 * communicate with the BIOS UEFI variables.
 */
#include <string.h>
#include "include/FSUefi.h"
#include "include/MPUefi.h"
#include "include/UefiVar.h"

#define REGISTRATION_COMPLETE_BIT_MASK 0x0001
#define PACKAGE_INFO_COMPLETE_BIT_MASK 0x0002

// defines for values verification
#define MP_VERIFY_UEFI_STRUCT_READ 1
#define MP_VERIFY_UEFI_VERSION_READ 1

MPUefi::MPUefi()
{
    string desiredUefiPath = string(EFIVARS_FILE_SYSTEM);
    m_uefi = new FSUefi(desiredUefiPath);
}

MpResult MPUefi::getRequestType(MpRequestType &type)
{
    MpResult res = MP_SUCCESS;
    size_t varDataSize = 0;
    SgxUefiVar *requestUefi = 0;

    do
    {
        requestUefi = (SgxUefiVar *)m_uefi->readUEFIVar(UEFI_VAR_SERVER_REQUEST, varDataSize);
        if (requestUefi == 0)
        {

            type = MP_REQ_NONE;
            break;
        }

        // structure version check
        if ((requestUefi->version != MP_BIOS_UEFI_VARIABLE_VERSION_1) &&
            (requestUefi->version != MP_BIOS_UEFI_VARIABLE_VERSION_2))
        {

            res = MP_UEFI_INTERNAL_ERROR;
            break;
        }

        if (varDataSize != sizeof(requestUefi->version) + sizeof(requestUefi->size) + requestUefi->size)
        {
            ;
            res = MP_UEFI_INTERNAL_ERROR;
            break;
        }

        // set request type
        if (0 == memcmp(requestUefi->header.guid, PlatformManifest_GUID, GUID_SIZE))
        {
            type = MP_REQ_REGISTRATION;
        }
        else if (0 == memcmp(requestUefi->header.guid, AddRequest_GUID, GUID_SIZE))
        {
            type = MP_REQ_ADD_PACKAGE;
        }
        else
        {
            res = MP_UEFI_INTERNAL_ERROR;
            break;
        }
    } while (0);

    if (requestUefi)
    {
        delete[] (uint8_t *)requestUefi;
    }
    return res;
}

MpResult MPUefi::getRequest(uint8_t *request, uint16_t &requestSize)
{
    MpResult res = MP_SUCCESS;
    size_t varDataSize = 0;
    SgxUefiVar *requestUefi = NULL;

    do
    {
        requestUefi = (SgxUefiVar *)m_uefi->readUEFIVar(UEFI_VAR_SERVER_REQUEST, varDataSize);
        if (requestUefi == 0)
        {
            res = MP_NO_PENDING_DATA;
            break;
        }

        if ((MP_BIOS_UEFI_VARIABLE_VERSION_1 != requestUefi->version) &&
            (MP_BIOS_UEFI_VARIABLE_VERSION_2 != requestUefi->version))
        {
            res = MP_INVALID_PARAMETER;
            break;
        }

        if (varDataSize != sizeof(requestUefi->version) + sizeof(requestUefi->size) + requestUefi->size)
        {

            res = MP_UEFI_INTERNAL_ERROR;
            break;
        }

        if (request)
        {
            if (requestSize < requestUefi->size)
            {
                res = MP_USER_INSUFFICIENT_MEM;
            }
            else
            {
                memcpy(request, &(requestUefi->header), requestUefi->size);
            }
        }
        requestSize = requestUefi->size;
    } while (0);

    if (requestUefi)
    {
        delete[] (uint8_t *)requestUefi;
    }
    return res;
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
            res = MP_UEFI_INTERNAL_ERROR;
            break;
        }

        // structure version check
        if (statusUefi->version != MP_BIOS_UEFI_VARIABLE_VERSION_1)
        {
            res = MP_UEFI_INTERNAL_ERROR;
            break;
        }

        // uefi structure size check
        if (statusUefi->size != sizeof(statusUefi->status) + sizeof(statusUefi->errorCode))
        {
            res = MP_UEFI_INTERNAL_ERROR;
            break;
        }

        memset(&status, 0, sizeof(status));

        if (statusUefi->status & REGISTRATION_COMPLETE_BIT_MASK)
        {
            status.registrationStatus = MP_MACHINE_REGISTERED;
        }
        if (statusUefi->status & PACKAGE_INFO_COMPLETE_BIT_MASK)
        {
            status.packageInfoStatus = MP_MACHINE_REGISTERED;
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

        if (status.registrationStatus == MP_MACHINE_REGISTERED)
        {
            statusUefi.status |= REGISTRATION_COMPLETE_BIT_MASK;
        }
        if (status.packageInfoStatus == MP_MACHINE_REGISTERED)
        {
            statusUefi.status |= PACKAGE_INFO_COMPLETE_BIT_MASK;
        }

        statusUefi.errorCode = status.errorCode;

        // write registration status to uefi
        int numOfBytes = m_uefi->writeUEFIVar(UEFI_VAR_STATUS, (const uint8_t *)(&statusUefi), sizeof(statusUefi), false);
        if (numOfBytes != sizeof(statusUefi))
        {
            if (numOfBytes == -1)
            {
                res = MP_INSUFFICIENT_PRIVILEGES;
                break;
            }
            res = MP_UEFI_INTERNAL_ERROR;
            break;
        }
    } while (0);

    return res;
}

MPUefi::~MPUefi()
{
    if (NULL != m_uefi)
    {
        m_uefi = NULL;
    }
}
