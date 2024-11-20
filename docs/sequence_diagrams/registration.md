# Platform Registration Flow

## Status Code

The platform registration service keeps a status code described below.

- `00`: Platform registered
- `1X`: SGX Platform status
  - `10`: Single socket platform
  - `11`: UEFI variables not available 
  - `12`: Direct/Indirect Registration already performed (unknown which)
- `2X`: Direct Registration Status
  - `20`: Failed to connect to Intel RS
  - `21`: Invalid registration request
    - Status Code: 400 --- Error Code: InvalidRequestSyntax
    - Status Code: 400 --- Error Code: InvalidPlatformManifest
    - Status Code: 415
  - `22`: Invalid platform data
    - Status Code: 400 --- Error Code: InvalidOrRevokedPackage 
    - Status Code: 400 --- Error Code: PackageNotFound 
    - Status Code: 400 --- Error Code: IncompatiblePackage 
  - `23`: Platform root keys can no longer be cached. Indirect registration already performed
    - Status Code: 400 --- Error Code: CachedKeysPolicyViolation
  - `24`: Intel RS could not process the request
    - Status Code: 500
    - Status Code: 503
- `3X`
- `4X`
- `5X`: PCK Cert Status
  - `50`: PCK Cert issued by PCK Processor CA and no information about the cached platform root keys is available
  - `51`: Platform root keys not cached by the Intel RS (Indirect Registration); this operation mode is not supported
- `9X`: General errors
  - `90`: IO error; see logs
  - `99`: Unknown or not supported error; see logs

## Sequence Diagrams

### 1. Main Flow

```mermaid
sequenceDiagram
    actor admin as Platform Admin
    participant w_cc_ipr as Wrapper CC Intel Platform Registration
    participant cc_ipr as CC Intel Platform Registration
    
    autonumber

    admin->>+w_cc_ipr: start
        note right of w_cc_ipr: TODO: how do we want to do this? Docker?

        w_cc_ipr->>w_cc_ipr: Read CC_IPR_REGISTRATION_INTERVAL
        note right of w_cc_ipr: Interval in minutes

        loop True
            w_cc_ipr->>+cc_ipr: register_platform()
                note right of cc_ipr: See diagram `2. Registration Flow`
            cc_ipr-->>-w_cc_ipr: Status Code

            w_cc_ipr->>w_cc_ipr: Print the status code to stderr

            w_cc_ipr->>w_cc_ipr: Wait for configured interval
        end

    deactivate w_cc_ipr
```

### 2. Registration Flow

```mermaid
sequenceDiagram
    participant cc_ipr as CC Intel Platform Registration
    participant pcs as SGX Platform Certification Service (PCS)

    autonumber

    activate cc_ipr

    cc_ipr->>cc_ipr: Read the number of CPU sockets in the platforms
        
    opt Number of socket is less than or equal to 1
        cc_ipr->>cc_ipr: Return status code 10
    end 

    cc_ipr->>cc_ipr: Read UEFI variable SgxRegistrationStatus

    opt UEFI variable SgxRegistrationStatus does NOT exist
        cc_ipr->>cc_ipr: Return status code 11
    end

    alt Flag SgxRegistrationStatus.SgxRegistrationComplete is UNSET 
        opt UEFI variable SgxRegistrationServerRequest does NOT exist
            cc_ipr->>cc_ipr: Return status code 11
        end

        cc_ipr->>cc_ipr: Register platform
        note right of cc_ipr: See diagram `2.2 Registration`

        opt If status code is different from 0
            cc_ipr->>cc_ipr: Return status code
        end
    else Flag SgxRegistrationStatus.SgxRegistrationComplete is SET
        opt Cached PCK Cert does NOT exist
           cc_ipr->>cc_ipr: Read the Encrypted PPID

            cc_ipr->>cc_ipr: Read the TCB Info

            cc_ipr-->>+pcs: GET https://api.trustedservices.intel.com/sgx/certification/v4/pckcert (body: Encrypted PPID, TCB Info)
            note right of cc_ipr: Returns the PCK Cert if the Intel RS has cached the platform root keys

            alt HTTP Status Code 200
                cc_ipr->>cc_ipr: Cache retrieved PCK Cert
                note right of cc_ipr: See diagram `2.1. Cache PCK Cert`

                opt If status code is different from 0
                    cc_ipr->>cc_ipr: Return status code
                end
            else
                cc_ipr->>cc_ipr: Return status code 12
                note right of cc_ipr: At this point we cannot determine if the Direct or Indirect Registration has been performed
            end
        end

        
    end

    note right of cc_ipr: TODO: Prometheus integration
    cc_ipr->>cc_ipr: Return status code 00

    deactivate cc_ipr
```

#### 2.1. Cache PCK Cert

```mermaid
sequenceDiagram
    participant cc_ipr as CC Intel Platform Registration

    autonumber

    activate cc_ipr

    note right of cc_ipr: Input: PCK Cert

    alt PCK Cert Cached Keys Flag does NOT exist
        cc_ipr->>cc_ipr: Return status code 50 
    else PCK Cert Cached Keys Flag is NOT set
        cc_ipr->>cc_ipr: Return status code 51 
    end

    cc_ipr->>cc_ipr: Cache retrieved PCK Cert

    alt IO error
        cc_ipr->>cc_ipr: Return status code 90 
    else
        cc_ipr->>cc_ipr: Return status code 00 
    end

    deactivate cc_ipr
```

#### 2.2 Registration

```mermaid
sequenceDiagram
    participant cc_ipr as CC Intel Platform Registration
    participant rs as Intel Registration Service (RS)

    autonumber

    activate cc_ipr
    
    cc_ipr->>cc_ipr: Read UEFI variable SgxRegistrationServerRequest
    note right of cc_ipr: The platform Manifest is available in that variable

    cc_ipr->>+rs: POST https://api.trustedservices.intel.com/sgx/registration/v1/platform (body: Platform Manifest)
        note right of cc_ipr: Direct registration: Key Caching Policy will be set to always<br> store platform root keys for the given platform 
    rs-->>-cc_ipr: PCK Cert Chain
    
    Alt PCK Cert Chain Downloaded
        cc_ipr->>cc_ipr: Cache retrieved PCK Cert
        note right of cc_ipr: See diagram `2.1. Cache PCK Cert`

        opt If status code is different from 0
            cc_ipr->>cc_ipr: Return status code
        end

        cc_ipr->>cc_ipr: Read UEFI variable SgxRegistrationStatus

        cc_ipr->>cc_ipr: Set the flag SgxRegistrationStatus.SgxRegistrationComplete
        note right of cc_ipr: After a reboot, the BIOS stops providing the Platform Manifest 

        cc_ipr->>cc_ipr: Write UEFI variable SgxRegistrationStatus
    Else Connection timeout
        cc_ipr->>cc_ipr: Return status code 20
    Else Invalid registration request
        cc_ipr->>cc_ipr: Return status code 21
    Else Invalid platform data
        cc_ipr->>cc_ipr: Return status code 22
    Else Indirect registration already performed
        cc_ipr->>cc_ipr: Return status code 23
    Else Intel RS could not process the request
        cc_ipr->>cc_ipr: Return status code 24
    Else
        cc_ipr->>cc_ipr: Return status code 99
    End

    deactivate cc_ipr
```

## Artifacts

* *Platform manifest*: A BLOB which contains the platform root pub keys used to register the SGX platform with the Intel Registration Service
* *PPID*: Platform Provisioning ID
* *TCB Info*: Compound of SGX TCB state (CPUSVN), PCESVN, and PCEID
* *PCK Cert*: X.509 certificate binding the PCE's key pair to a certain SGX TCB state
* *PCK Cert Cached Keys Flag*: PCK Cert extension under OID `1.2.840.113741.1.13.1.7.2` to state whether the platform root keys are cached by Intel RS

## Documentation

- [Intel SGX DCAP Multipackage SW](https://download.01.org/intel-sgx/sgx-dcap/1.9/linux/docs/Intel_SGX_DCAP_Multipackage_SW.pdf)
- [SGX PCK Certificate Specification](https://download.01.org/intel-sgx/latest/dcap-latest/linux/docs/SGX_PCK_Certificate_CRL_Spec-1.4.pdf)