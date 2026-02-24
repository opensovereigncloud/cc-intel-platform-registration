######## SGX SDK Settings ########
SGX_DEBUG ?= 1

SGX_SDK ?= /opt/intel/sgxsdk
SGX_ARCH ?= x64
SGX_QPL_LOGGING ?= 1
SGX_TRACE_LOGGING ?= 0

SGX_COMMON_CFLAGS := -m64
SGX_LIBRARY_PATH := $(SGX_SDK)/lib64
SGX_ENCLAVE_SIGNER := $(SGX_SDK)/bin/x64/sgx_sign
SGX_EDGER8R := $(SGX_SDK)/bin/x64/sgx_edger8r

ifeq ($(SGX_DEBUG), 1)
        SGX_COMMON_CFLAGS += -O0 -g
else
        SGX_COMMON_CFLAGS += -O2
endif


######## Shared Library Settings ########
LIB_NAME := libsgx_platform_info.so
INCLUDE_DIR := include
ENCLAVE_DIR := /opt/cc-intel-platform-registration
BUILD_DIR = ../../build
LIB_DIR = $(BUILD_DIR)/lib


######## Library Names ########
DCAP_Library_Name := sgx_dcap_ql
Urts_Library_Name := sgx_urts
Pce_Library_Name := sgx_pce_logic
Trts_Library_Name := sgx_trts
Service_Library_Name := sgx_tservice
Crypto_Library_Name := sgx_tcrypto

# ######## sgx_platform_info Settings ########
sgx_platform_info_Cpp_Files := src/sgx_platform_info.cpp
sgx_platform_info_Include_Paths := -I$(SGX_SDK)/include

sgx_platform_info_C_Flags := $(SGX_COMMON_CFLAGS) -fPIC -Wno-attributes $(sgx_platform_info_Include_Paths)

ifeq ($(SGX_DEBUG), 1)
        sgx_platform_info_C_Flags += -DDEBUG -UNDEBUG -UEDEBUG
else
        sgx_platform_info_C_Flags += -DNDEBUG -UEDEBUG -UDEBUG
endif

sgx_platform_info_Cpp_Flags := $(sgx_platform_info_C_Flags) -std=c++11
sgx_platform_info_Link_Flags := $(SGX_COMMON_CFLAGS) -l$(DCAP_Library_Name) -l$(Urts_Library_Name) -l$(Pce_Library_Name) -ldl -lpthread

sgx_platform_info_Cpp_Objects := $(sgx_platform_info_Cpp_Files:.cpp=.o)
sgx_platform_info_Name := sgx_platform_info

######## Enclave Settings ########
Enclave_Cpp_Files := Enclave/Enclave.cpp
Enclave_Include_Paths := -I$(SGX_SDK)/include -I$(SGX_SDK)/include/tlibc -I$(SGX_SDK)/include/libcxx

CC_BELOW_4_9 := $(shell expr "`$(CC) -dumpversion`" \< "4.9")
ifeq ($(CC_BELOW_4_9), 1)
        Enclave_C_Flags := $(SGX_COMMON_CFLAGS) -nostdinc -fvisibility=hidden -fpie -ffunction-sections -fdata-sections -fstack-protector
else
        Enclave_C_Flags := $(SGX_COMMON_CFLAGS) -nostdinc -fvisibility=hidden -fpie -ffunction-sections -fdata-sections -fstack-protector-strong
endif

Enclave_C_Flags += $(Enclave_Include_Paths)
Enclave_Cpp_Flags := $(Enclave_C_Flags) -std=c++11 -nostdinc++

Enclave_Link_Flags := $(SGX_COMMON_CFLAGS) -Wl,--no-undefined -nostdlib -nodefaultlibs -nostartfiles -L$(SGX_LIBRARY_PATH) \
        -Wl,--whole-archive -l$(Trts_Library_Name) -Wl,--no-whole-archive \
        -Wl,--start-group -lsgx_tstdc -l$(Crypto_Library_Name) -l$(Service_Library_Name) -lsgx_tstdc -Wl,--end-group \
        -Wl,-Bstatic -Wl,-Bsymbolic -Wl,--no-undefined \
        -Wl,-pie,-eenclave_entry -Wl,--export-dynamic  \
        -Wl,--defsym,__ImageBase=0 -Wl,--gc-sections   \
        -Wl,-z,relro,-z,now,-z,noexecstack             \
        -Wl,--version-script=Enclave/Enclave.lds

Enclave_Cpp_Objects := $(Enclave_Cpp_Files:.cpp=.o)
Enclave_Name := sgx_platform_enclave.so
Signed_Enclave_Name := sgx_platform_enclave.signed.so
Enclave_Config_File := Enclave/Enclave.config.xml
Enclave_Test_Key := Enclave/Enclave_private_sample.pem



######## Build Flags ########
ifeq ($(SGX_DEBUG), 1)
        Build_Mode = HW_DEBUG
else
        Build_Mode = HW_RELEASE
endif

.PHONY: all clean
all: .config_$(Build_Mode)_$(SGX_ARCH) enclave shared_lib cp2base

.config_$(Build_Mode)_$(SGX_ARCH):
	@rm -f .config_* $(sgx_platform_info_Name) $(Enclave_Name) $(Signed_Enclave_Name) $(sgx_platform_info_Cpp_Objects) src/Enclave_u.* $(Enclave_Cpp_Objects) Enclave/Enclave_t.*
	@touch .config_$(Build_Mode)_$(SGX_ARCH)

######## Enclave Objects ########
enclave: $(Signed_Enclave_Name)

Enclave/Enclave_t.c: Enclave/Enclave.edl
	@cd Enclave && $(SGX_EDGER8R) --trusted ../Enclave/Enclave.edl
	@echo "GEN  =>  $@"

Enclave/Enclave_t.o: Enclave/Enclave_t.c
	@$(CC) $(Enclave_C_Flags) -c $< -o $@
	@echo "CC   <=  $<"

$(Enclave_Cpp_Objects): Enclave/Enclave_t.c
Enclave/%.o: Enclave/%.cpp
	@$(CXX) $(Enclave_Cpp_Flags) -c $< -o $@
	@echo "CXX  <=  $<"

$(Enclave_Name): Enclave/Enclave_t.o $(Enclave_Cpp_Objects)
	@$(CXX) $^ -o $@ $(Enclave_Link_Flags)
	@echo "LINK =>  $@"

$(Signed_Enclave_Name): $(Enclave_Name)
ifeq ($(wildcard $(Enclave_Test_Key)),)
	@echo "There is no enclave test key<Enclave_private_sample.pem>."
	@echo "The project will generate a key<Enclave_private_sample.pem> for test."
	@openssl genrsa -out $(Enclave_Test_Key) -3 3072
endif
	@$(SGX_ENCLAVE_SIGNER) sign -key $(Enclave_Test_Key) -enclave $(Enclave_Name) -out $@ -config $(Enclave_Config_File)
	@echo "SIGN =>  $@"

# ######## sgx_platform_info Objects ########
sgx_platform_info: $(sgx_platform_info_Name)

src/Enclave_u.c: Enclave/Enclave.edl
	@cd src && $(SGX_EDGER8R) --untrusted ../Enclave/Enclave.edl
	@echo "GEN  =>  $@"

src/Enclave_u.o: src/Enclave_u.c
	@$(CC) $(sgx_platform_info_C_Flags) -c $< -o $@
	@echo "CC   <=  $<"

$(sgx_platform_info_Cpp_Objects): src/Enclave_u.c
src/%.o: src/%.cpp
	@$(CXX) $(sgx_platform_info_Cpp_Flags) -c $< -o $@
	@echo "CXX  <=  $<"

$(sgx_platform_info): src/Enclave_u.o $(sgx_platform_info_Cpp_Objects)
	$(CXX) $^ -o $@ $(sgx_platform_info_Link_Flags)
	@cp $(sgx_platform_info) $(LIB_DIR)/
	@echo "LINK =>  $@"



######## Shared Library ########
.PHONY: shared_lib
shared_lib: dirs $(LIB_DIR)/$(LIB_NAME)

dirs:
	@mkdir -p $(INCLUDE_DIR) $(LIB_DIR)

cp2base:
	@cp $(LIB_DIR)/*.so /usr/local/lib


Lib_Objects := src/Enclave_u.o $(sgx_platform_info_Cpp_Objects) $(Enclave_Cpp_Objects)
Lib_Link_Flags := -shared $(SGX_COMMON_CFLAGS) \
    -L$(SGX_LIBRARY_PATH) \
    -l$(DCAP_Library_Name) \
    -l$(Urts_Library_Name) \
    -l$(Pce_Library_Name) \
    -l$(Trts_Library_Name) \
    -l$(Service_Library_Name) \
    -l$(Crypto_Library_Name) \
    -lsgx_tstdc \
    -ldl -lpthread

dirs:
	@mkdir -p $(INCLUDE_DIR) $(LIB_DIR) $(ENCLAVE_DIR)

$(LIB_DIR)/$(LIB_NAME): src/Enclave_u.o $(sgx_platform_info_Cpp_Objects)  
	$(CXX) $(sgx_platform_info_Cpp_Flags) -shared -o $@ $^ $(sgx_platform_info_Link_Flags)
	@cp $(Signed_Enclave_Name) $(LIB_DIR)/
	@cp $(Signed_Enclave_Name) $(ENCLAVE_DIR)/


clean:
	@rm -f $(sgx_platform_info_Name) $(Enclave_Name) $(Signed_Enclave_Name) $(Enclave_Test_Key)
	@rm -f src/Enclave_u.* src/*.o
	@rm -f Enclave/Enclave_t.* Enclave/*.o
	@rm -f .config_*
	@rm -rf $(LIB_DIR) $(INCLUDE_DIR)
	
