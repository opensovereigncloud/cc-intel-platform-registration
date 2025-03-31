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
 * File: mp_uefi.h
 *
 * Description: C definition of function wrappers for C++ methods in
 * the MPUefi class implementation.
 */
#ifndef MP_UEFI_H
#define MP_UEFI_H

#include "MultiPackageDefs.h"
#include <MPUefi.h>

/**
 * This is the main entry point for the Multi-Package UEFI CPP interface.
 * Used to get and set various UEFI variables for the Multi-Package registration flows.
 */

extern "C"
{
    /**
     * Multi-Package UEFI interface initiation.
     *
     * @return status code, one of:
     *      - MP_SUCCESS
     *      - MP_INVALID_PARAMETER
     *      - MP_REDUNDANT_OPERATION
     *      - MP_MEM_ERROR
     */
    MpResult mp_uefi_init();

    /**
     * Retrieves the pending request type.
     * The BIOS generates a request when there is a pending request to be sent to the SGX Registration Server.
     *
     * @param type - output parameter, holds the pending request type or MP_REQ_NONE.
     *
     * @return status code, one of:
     *      - MP_SUCCESS
     *      - MP_INVALID_PARAMETER
     *      - MP_UEFI_INTERNAL_ERROR
     */
    MpResult mp_uefi_get_request_type(MpRequestType *type);

    /**
     * Retrieves the content of a pending request.
     * The request should be sent to the SGX Registration Server.
     * The BIOS generates PlatformManifest request for first "Platform Binding" and "TCB Recovery".
     * The BIOS generates AddPackage request to register a new "Added Package".
     *
     * @param request       - output parameter, holds the request buffer to be populated.
     * @param request_size  - input parameter, size of request buffer in bytes.
     *                      - output paramerter, holds the actual size written to request buffer.
     *                        if response equals MP_USER_INSUFFICIENT_MEM or if request buffer is NULL, holds the pending request size.
     *
     * @return status code, one of:
     *      - MP_SUCCESS
     *      - MP_INVALID_PARAMETER
     *      - MP_NO_PENDING_DATA
     *      - MP_USER_INSUFFICIENT_MEM
     *      - MP_UEFI_INTERNAL_ERROR
     */
    MpResult mp_uefi_get_request(uint8_t *request, uint16_t *request_size);

    /**
     * Retrieves the current registration status.
     *
     * @param status    - output parameter, holds the current registration status.
     *
     * @return status code, one of:
     *      - MP_SUCCESS
     *      - MP_INVALID_PARAMETER
     *      - MP_UEFI_INTERNAL_ERROR
     */

    MpResult mp_uefi_get_registration_status(MpRegistrationStatus *status);

    /**
     * Sets the registration status.
     *
     * @param status    - input parameter, holds the desired registration status.
     *
     * @return status code, one of:
     *      - MP_SUCCESS
     *      - MP_INVALID_PARAMETER
     *      - MP_UEFI_INTERNAL_ERROR
     */
    MpResult mp_uefi_set_registration_status(const MpRegistrationStatus *status);

    /**
     * Multi-Package UEFI interface termination.
     *
     * @return status code, one of:
     *      - MP_SUCCESS
     *      - MP_REDUNDANT_OPERATION
     */
    MpResult mp_uefi_terminate();
};
#endif // #ifndef MP_UEFI_H
