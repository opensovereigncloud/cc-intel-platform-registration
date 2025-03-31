# Compiler and flags
CXX = g++
AR = ar
CXXFLAGS = -Wall -Wextra -fPIC -std=c++17 -Isrc/include -Isrc/include/c_wrapper
LDFLAGS = -Llib 
ARFLAGS = rcs


# Directories
BUILD_DIR = ../../build
SRC_DIR = src
OBJ_DIR = $(BUILD_DIR)/mp_management/obj
LIB_DIR = $(BUILD_DIR)/lib

# Source files (excluding mp_management.cpp and mp_uefi.cpp)
COMMON_SRCS = $(filter-out $(SRC_DIR)/c_wrapper/mp_management.cpp $(SRC_DIR)/c_wrapper/mp_uefi.cpp, \
              $(wildcard $(SRC_DIR)/*.cpp) \
              $(wildcard $(SRC_DIR)/c_wrapper/*.cpp))

# Library specific source files
MP_MGMT_SRCS = $(SRC_DIR)/c_wrapper/mp_management.cpp
MP_UEFI_SRCS = $(SRC_DIR)/c_wrapper/mp_uefi.cpp

# Object files
COMMON_OBJS = $(COMMON_SRCS:$(SRC_DIR)/%.cpp=$(OBJ_DIR)/%.o)
MP_MGMT_OBJS = $(MP_MGMT_SRCS:$(SRC_DIR)/%.cpp=$(OBJ_DIR)/%.o)
MP_UEFI_OBJS = $(MP_UEFI_SRCS:$(SRC_DIR)/%.cpp=$(OBJ_DIR)/%.o)

# Library names
MP_MGMT_LIB = $(LIB_DIR)/libmp_management.so

# Targets
.PHONY: all clean directories

all: directories $(MP_MGMT_LIB) $(MP_UEFI_LIB)

# Create necessary directories
directories:
	@mkdir -p $(OBJ_DIR)/c_wrapper
	@mkdir -p $(LIB_DIR)

# shared library (mp_management)
$(MP_MGMT_LIB): $(COMMON_OBJS) $(MP_MGMT_OBJS)
	$(CXX) -shared -o $@ $^ $(LDFLAGS)

# Compile source files
$(OBJ_DIR)/%.o: $(SRC_DIR)/%.cpp
	@mkdir -p $(dir $@)
	$(CXX) $(CXXFLAGS) -c $< -o $@


# Clean
clean:
	rm -rf $(OBJ_DIR) $(LIB_DIR) 

# Print variables for debugging
print-%:
	@echo $* = $($*)
	
