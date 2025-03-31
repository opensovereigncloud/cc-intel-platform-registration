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
 * File: MPManagement.cpp
 *
 * Description: Implemenation of the MPManagement class.  These are the
 * methods that will read and write to the UEFI variables to provide
 * the ability to configure, collect data and collect status of the
 * SGX MP platform.
 */
#include <string.h>
#include "include/MPUefi.h"
#include "include/MPManagement.h"

MPManagement::MPManagement()
{
    m_mpuefi = new MPUefi();
}

MpResult MPManagement::getRegistrationStatus(MpMachineRegistrationStatus &status)
{
    MpRegistrationStatus regStatus;
    MpResult res = MP_UNEXPECTED_ERROR;

    do
    {
        res = m_mpuefi->getRegistrationStatus(regStatus);
        if (MP_SUCCESS != res)
        {
            break;
        }

        if (regStatus.registrationStatus)
        {
            status = MP_MACHINE_REGISTERED;
        }
        else
        {
            status = MP_MACHINE_NOT_REGISTERED;
        }

        res = MP_SUCCESS;
    } while (0);

    return res;
}

MpResult MPManagement::setRegistrationStatusAsComplete()
{
    MpRegistrationStatus regStatus;
    MpResult res = MP_UNEXPECTED_ERROR;

    do
    {
        res = m_mpuefi->getRegistrationStatus(regStatus);
        if (MP_SUCCESS != res)
        {
            break;
        }
        regStatus.registrationStatus = MP_MACHINE_REGISTERED;
        res = m_mpuefi->setRegistrationStatus(regStatus);
        if (MP_SUCCESS != res)
        {
            break;
        }
        res = MP_SUCCESS;
    } while (0);

    return res;
}

MpResult MPManagement::getRequestData(uint8_t *buffer, uint16_t &buffer_size, MpRequestType expectedRequestType)
{
    MpRegistrationStatus status;
    MpRequestType type;
    MpResult res = MP_UNEXPECTED_ERROR;
    string expectedArtifactTypeName = (expectedRequestType == MpRequestType::MP_REQ_REGISTRATION) ? "PlatformManifest" : "ADD_REQUEST";
    string functionName = (expectedRequestType == MpRequestType::MP_REQ_REGISTRATION) ? "getPlatformManifest" : "getAddPackageRequest";

    do
    {
        if (NULL == buffer)
        {
            res = MP_INVALID_PARAMETER;
            break;
        }

        res = m_mpuefi->getRegistrationStatus(status);
        if (MP_SUCCESS != res)
        {

            break;
        }

        if (MP_MACHINE_NOT_REGISTERED != status.registrationStatus)
        {
            res = MP_NO_PENDING_DATA;
            break;
        }

        res = m_mpuefi->getRequestType(type);
        if (MP_SUCCESS != res)
        {
            break;
        }

        if (expectedRequestType != type)
        {
            res = MP_NO_PENDING_DATA;
            break;
        }

        uint8_t requestData[MAX_REQUEST_SIZE];
        uint16_t size = sizeof(requestData);
        memset(&requestData, 0, sizeof(requestData));
        res = m_mpuefi->getRequest((uint8_t *)&requestData, size);
        if (MP_SUCCESS != res)
        {
            if (MP_NO_PENDING_DATA == res)
            {
                res = MP_UEFI_INTERNAL_ERROR;
            }
            break;
        }
        if (buffer_size < size)
        {
            buffer_size = size;
            res = MP_USER_INSUFFICIENT_MEM;
            break;
        }

        memcpy(buffer, &requestData, size);
        buffer_size = size;
        if (MP_SUCCESS != res)
        {
            break;
        }
        res = MP_SUCCESS;
    } while (0);
    return res;
}

MpResult MPManagement::getPlatformManifest(uint8_t *buffer, uint16_t &buffer_size)
{
    return getRequestData(buffer, buffer_size, MP_REQ_REGISTRATION);
}

MPManagement::~MPManagement()
{
    if (NULL != m_mpuefi)
    {
        delete m_mpuefi;
        m_mpuefi = NULL;
    }
}
