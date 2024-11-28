# Platform Registration Flow

## Limitation

- Addition or replacement of CPUs is not supported.

## Status Code

The platform registration service keeps a status code described below.

- `0X`: SGX status
  - `00`: Platform directly registered
  - `01`: Platform indirectly registered
  - `02`: Pending execution
  - `03`: SGX UEFI variables not available 
  - `04`: Direct/Indirect Registration already performed (unknown which)
  - `05`: Platform reboot required
- `1X`: Registration Status
  - `10`: Failed to connect to Intel RS
  - `11`: Invalid registration request
    - Status Code: 400 --- Error Code: InvalidRequestSyntax
    - Status Code: 400 --- Error Code: InvalidPlatformManifest
    - Status Code: 415
  - `12`: Invalid platform data
    - Status Code: 400 --- Error Code: InvalidOrRevokedPackage 
    - Status Code: 400 --- Error Code: PackageNotFound 
    - Status Code: 400 --- Error Code: IncompatiblePackage 
  - `13`: Platform root keys can no longer be cached. Indirect registration already performed
    - Status Code: 400 --- Error Code: CachedKeysPolicyViolation
  - `14`: Intel RS could not process the request
    - Status Code: 500
    - Status Code: 503
- `5X`: PCK Cert Status
  - `50`: Invalid PCK Cert
  - `51`: Not this platform PCK Cert
  - `52`: PCK Cert issued by PCK Processor CA and no information about the cached platform root keys is available
  - `53`: Platform root keys not cached by the Intel RS (Indirect Registration); this operation mode is not supported
- `9X`: General errors
  - `99`: Unknown or not supported error; see logs

## Sequence Diagrams

### 1. Main Flow

```mermaid
sequenceDiagram
    actor admin as Platform Admin
    participant cc_ipr as CC Intel Platform Registration
    
    autonumber

    admin->>+cc_ipr: launch
        note right of cc_ipr: TODO: how do we want to do this? Docker?

        cc_ipr->>cc_ipr: Read CC_IPR_REGISTRATION_INTERVAL
        note right of cc_ipr: Interval in minutes

        cc_ipr->>cc_ipr: Initialize status code with the value of 02

        rect rgb(100, 200, 100)
            note right of cc_ipr: This block is spawned
            
            loop True
                cc_ipr->>+cc_ipr: register_platform()
                    note right of cc_ipr: See diagram `2. Registration Check Flow`
                cc_ipr-->>-cc_ipr: Status Code

                cc_ipr->>cc_ipr: Update the Prometheus Metric with the status code value

                cc_ipr->>cc_ipr: Wait for configured interval
            end
        end

        cc_ipr->>cc_ipr: Launch the Prometheus Metrics Server
        note right of cc_ipr: Status code available in path `/metrics`

    deactivate cc_ipr
```

### 2. Registration Check Flow

```mermaid
sequenceDiagram
    participant cc_ipr as CC Intel Platform Registration
    participant pcs as SGX Platform Certification Service (PCS)

    autonumber

    activate cc_ipr

    cc_ipr->>cc_ipr: Read UEFI variable SgxRegistrationStatus

    opt UEFI variable SgxRegistrationStatus does NOT exist
        cc_ipr->>cc_ipr: Return status code 03
    end

    alt Flag SgxRegistrationStatus.SgxRegistrationComplete is UNSET 
        cc_ipr->>cc_ipr: Read UEFI variable SgxRegistrationServerRequest
        note right of cc_ipr: The Platform Manifest is available in that variable

        opt UEFI variable SgxRegistrationServerRequest does NOT exist
            cc_ipr->>cc_ipr: Return status code 03
        end

        cc_ipr->>cc_ipr: Register platform(Platform Manifest)
        note right of cc_ipr: See diagram `2.2. Registration`

        cc_ipr->>cc_ipr: Return status code

    else Flag SgxRegistrationStatus.SgxRegistrationComplete is SET
        opt UEFI variable SgxRegistrationServerRequest exists
            note right of cc_ipr: We finished the registration and updated SgxRegistrationStatus.SgxRegistrationComplete.<br> However, the reboot has not been performed yet so that<br> the BIOS can remove SgxRegistrationServerRequest
            cc_ipr->>cc_ipr: Return status code 05
        end

        note right of cc_ipr: We want to determine whether Direct or Indirect Registration was performed
        cc_ipr->>cc_ipr: Read the Encrypted PPID

        cc_ipr->>cc_ipr: Read the PCEID

        cc_ipr->>+pcs: GET https://api.trustedservices.intel.com/sgx/certification/v4/pckcert (body: Encrypted PPID, PCEID)
            note right of cc_ipr: Returns the PCK Cert if the Intel RS has cached the platform root keys (aka. Direct Registration)
        pcs-->>-cc_ipr: PCK Cert Chain

        alt HTTP Status Code 200
            cc_ipr->>cc_ipr: Validate retrieved PCK Cert
            note right of cc_ipr: See diagram `2.1. Validate PCK Cert`

            cc_ipr->>cc_ipr: Return status code
        else HTTP Status Code 404
            note right of cc_ipr: UEFI SGX variables available, SGX registration set as Complete and PCK Cert not found
            cc_ipr->>cc_ipr: Return status code 01
        else
            cc_ipr->>cc_ipr: Return status code 04
            note right of cc_ipr: At this point we cannot determine if the Direct or Indirect Registration has been performed
        end     
    end

    deactivate cc_ipr
```

#### 2.1. Validate PCK Cert

```mermaid
sequenceDiagram
    participant cc_ipr as CC Intel Platform Registration

    autonumber

    activate cc_ipr

    note right of cc_ipr: Input: PCK Cert

    cc_ipr->>cc_ipr: Read PPID

    alt Failed to parse PCK Cert
        cc_ipr->>cc_ipr: Return status code 50
    else PPID does NOT match
        cc_ipr->>cc_ipr: Return status code 51
    else PCK Cert Cached Keys Flag does NOT exist
        cc_ipr->>cc_ipr: Return status code 52 
    else PCK Cert Cached Keys Flag is NOT set
        cc_ipr->>cc_ipr: Return status code 53
    else
        cc_ipr->>cc_ipr: Return status code 00 
    end

    deactivate cc_ipr
```

#### 2.2. Registration

To set the `Key Caching Policy` to true, we **must** register the Platform with Intel Registration Service first.
This service will then store the Platform Root Keys.

The flow below also supports `TCB Recovery` and `SGX Reset`.

```mermaid
sequenceDiagram
    participant cc_ipr as CC Intel Platform Registration
    participant rs as Intel Registration Service (RS)

    autonumber

    activate cc_ipr
    note right of cc_ipr: Input: Platform Manifest

    cc_ipr->>+rs: POST https://api.trustedservices.intel.com/sgx/registration/v1/platform (body: Platform Manifest)
        note right of cc_ipr: Direct registration: Key Caching Policy will be set to always<br> store platform root keys for the given platform
    rs-->>-cc_ipr: PCK Cert Chain
    
    Alt Operation successful
        cc_ipr->>cc_ipr: Validate retrieved PCK Cert
        note right of cc_ipr: See diagram `2.1. Validate PCK Cert`

        opt If status code is different from 0
            cc_ipr->>cc_ipr: Return status code
        end

        cc_ipr->>cc_ipr: Read UEFI variable SgxRegistrationStatus

        cc_ipr->>cc_ipr: Set the flag SgxRegistrationStatus.SgxRegistrationComplete
        note right of cc_ipr: After a reboot, the BIOS stops providing the Platform Manifest 

        cc_ipr->>cc_ipr: Write UEFI variable SgxRegistrationStatus

        cc_ipr->>cc_ipr: Return status code 05
    Else Connection timeout
        cc_ipr->>cc_ipr: Return status code 10
    Else Invalid registration request
        cc_ipr->>cc_ipr: Return status code 11
    Else Invalid platform data
        cc_ipr->>cc_ipr: Return status code 12
    Else Indirect registration already performed
        cc_ipr->>cc_ipr: Return status code 13
    Else Intel RS could not process the request
        cc_ipr->>cc_ipr: Return status code 14
    Else
        cc_ipr->>cc_ipr: Return status code 99
    End

    deactivate cc_ipr
```

## Artifacts

* *Platform manifest*: A BLOB which contains the platform root pub keys used to register the SGX platform with the Intel Registration Service
* *PPID*: Unique Platform Provisioning ID of the processor package or platform instance used by Provisioning Certification Enclave. The PPID does not depend on the TCB.
* *PCEID*: Identifier of the Intel SGX enclave that uses Provisioning Certification Key to sign proofs that attestation keys or attestation key provisioning protocol messages are created on genuine hardware
* *PCK Cert*: X.509 certificate binding the PCE's key pair to a certain SGX TCB state
* *PCK Cert Cached Keys Flag*: PCK Cert extension under OID `1.2.840.113741.1.13.1.7.2` to state whether the platform root keys are cached by Intel RS

## Documentation

- [Intel SGX DCAP Multipackage SW](https://download.01.org/intel-sgx/sgx-dcap/1.9/linux/docs/Intel_SGX_DCAP_Multipackage_SW.pdf)
- [SGX PCK Certificate Specification](https://download.01.org/intel-sgx/latest/dcap-latest/linux/docs/SGX_PCK_Certificate_CRL_Spec-1.4.pdf)