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
 * File: MPManagement.h
 *
 * Description: Classe definition for the management tool functionality.
 */
#ifndef __MPMANAGEMENT_H
#define __MPMANAGEMENT_H

#include <string>
#include "MultiPackageDefs.h"

class MPUefi;

/**
 * This is the main entry point for the Multi-Package Management interface library.
 * Used to manage the registration SGX feature and registration flow.
 */

class MPManagement
{
public:
    MPManagement();

    // Retrieves PlatformManifest.
    // if PlatformManifest UEFI are ready for reading, copies the PlatformManifest to input buffer.
    // if not, returns an appropriate error (MP_NO_PENDING_DATA). populates buffer_size with the required size in case of insufficient size.
    virtual MpResult getPlatformManifest(uint8_t *buffer, uint16_t &buffer_size);

    // Retrieves registration error code.
    // If registration is completed successfully, error_code will be set to 0.
    // If registration process failed, error_code will be set to the relevant last reported error code.
    virtual MpResult getRegistrationStatus(MpMachineRegistrationStatus &status);

    // Sets the machine registration status to completed.
    virtual MpResult setRegistrationStatusAsComplete();

    virtual ~MPManagement();

private:
    MPUefi *m_mpuefi;

    MPManagement &operator=(const MPManagement &) { return *this; }
    MPManagement(const MPManagement &src) { (void)src; }
    MpResult getRequestData(uint8_t *buffer, uint16_t &buffer_size, MpRequestType requestType);
};

#endif // #ifndef __MPMANAGEMENT_H
