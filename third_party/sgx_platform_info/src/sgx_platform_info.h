#ifndef _SGX_PLATFORM_INFO_H
#define _SGX_PLATFORM_INFO_H
#ifdef __cplusplus
extern "C"
{
#endif
#define MAX_ENCRYPTED_PPID_SIZE 384
#define ENCLAVE_PATH "/opt/cc-intel-platform-registration/sgx_platform_enclave.signed.so"
#include "sgx_pce.h"

#define GET_PLATFORM_MK_ERROR(x) (0x0000F000 | (x))
    typedef enum _get_plaform_error_t
    {
        /* Enclave creation Failed */
        ENCLAVE_CREATE_FAIL = GET_PLATFORM_MK_ERROR(0x0009),
    } get_plaform_error_t;

    typedef struct _platform_info_t
    {
        sgx_pce_info_t pce_info;
        uint32_t encrypted_ppid_out_size;
        uint8_t encrypted_ppid[MAX_ENCRYPTED_PPID_SIZE];
    } platform_info_t;
    /*

    Get Platform Info:
    Runs an enclave and return all the platform info, including the encrypted PPID
        The PPID is encrypted using the INTEL PPIDEK
    Params:
        [OUT]: Platform_info_t
    */
    u_int32_t get_platform_info(platform_info_t *platform_info);

#ifdef __cplusplus
}
#endif
#endif