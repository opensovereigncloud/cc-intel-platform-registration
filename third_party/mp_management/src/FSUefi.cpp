/*
 * Copyright(c) 2011-2025 Intel Corporation
 *
 * SPDX-License-Identifier: BSD-3-Clause
 */

#include <sys/stat.h>
#include <sys/ioctl.h>
#include <linux/fs.h>
#include <fcntl.h>
#include <errno.h>
#include <string.h>
#include <type_traits>
#include <unistd.h>
#include "FSUefi.h"
#include "logger.h"

static constexpr uint8_t attributes[4] = {0x07, 0x00, 0x00, 0x00};

namespace
{

    template <
        typename T,
        typename std::enable_if<std::is_integral<T>::value, int>::type = 0>
    T *allocArray(size_t count)
    {
        try
        {
            return new T[count];
        }
        catch (const std::bad_alloc &)
        {
            return nullptr;
        }
    }

} // namespace

long FSUefi::fdGetVarFileSize(int fd)
{
    struct stat stat_buf;
    int rc = fstat(fd, &stat_buf);
    return rc == 0 ? stat_buf.st_size : -1;
}

uint8_t *FSUefi::readUEFIVar(const char *varName, size_t &dataSize)
{
    uint8_t *entire_var = NULL;
    uint8_t *var_data = NULL;
    int fd = -1;

    // get abs uefi path
    const std::string fullPath = m_uefiAbsPath + varName;

    do
    {
        errno = 0;
        fd = open(fullPath.c_str(), O_RDONLY);
        if (fd == -1)
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "readUEFIVar: failed to open uefi variable %s ,error: %s\n", fullPath.c_str(), strerror(errno));
            break;
        }

        // get uefi file size
        const long tempSize = fdGetVarFileSize(fd);
        if (tempSize < 0)
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "readUEFIVar: failed to get variable file size %ld \n", tempSize);
            break;
        }

        const size_t fileSize = static_cast<size_t>(tempSize);
        // actual data size without uefi attribute
        dataSize = fileSize - sizeof(attributes);

        entire_var = allocArray<uint8_t>(fileSize);
        if (!entire_var)
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "readUEFIVar: allocation failure\n");
            break;
        }

        errno = 0;
        ssize_t bytesRead = read(fd, entire_var, fileSize);
        if (bytesRead <= 0)
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "readUEFIVar: failed to read uefi variable %s ,error: %s\n", fullPath.c_str(), strerror(errno));
            break;
        }

        var_data = allocArray<uint8_t>(dataSize);
        if (!var_data)
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "readUEFIVar: allocation failure\n");
            break;
        }

        memcpy(var_data, entire_var + sizeof(attributes), dataSize);
    } while (0);

    if (entire_var)
    {
        delete[] entire_var;
    }
    if (fd != -1)
    {
        close(fd);
    }
    return var_data;
}

int FSUefi::writeUEFIVar(const char *varName, const uint8_t *data, size_t dataSize, bool create)
{
    int fd = -1;
    int rc = -1;
    int flags, oflags = 0;
    mode_t mode = 0;
    ssize_t bytesWrote = 0;
    char *buffer = NULL;
    int lastOpenErrno = 0;

    do
    {
        if (dataSize + sizeof(attributes) < dataSize)
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: the data size is too big\n");
            break;
        }
        buffer = allocArray<char>(dataSize + sizeof(attributes));
        if (!buffer)
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: allocation failure\n");
            break;
        }

        // zero the buffer and copy data
        memset(buffer, 0, sizeof(attributes) + dataSize);
        memcpy(buffer, attributes, sizeof(attributes));
        memcpy(buffer + sizeof(attributes), data, dataSize);

        // get abs uefi path
        const std::string fullPath = m_uefiAbsPath + varName;

        // set read only flag
        flags = O_RDONLY;

        errno = 0;
        fd = open(fullPath.c_str(), flags);
        if (fd == -1)
        {
            if (ENOENT == errno)
            {
                lastOpenErrno = errno;
                if (create)
                {
                    // set flags: create, user r/w permission, group read permission, others read permission.
                    flags |= O_CREAT;
                    mode |= S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH;
                    fd = open(fullPath.c_str(), flags, mode);
                }
            }
            if (fd == -1)
            {
                if ((create) && (ENOENT == lastOpenErrno))
                {
                    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: failed to create uefi variable %s, error: %s\n", fullPath.c_str(), strerror(errno));
                }
                else
                {
                    uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: failed to open uefi variable %s, error: %s\n", fullPath.c_str(), strerror(errno));
                }
                break;
            }
        }
        else
        {
            // get uefi file size
            long tempSize = fdGetVarFileSize(fd);
            if (tempSize < 0)
            {
                uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: failed to get variable file size %ld \n", tempSize);
                break;
            }

            uint8_t uefiAttributes[4];
            errno = 0;
            ssize_t bytesRead = read(fd, uefiAttributes, 4);
            if (bytesRead != 4)
            {
                uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: failed to read uefi variable %s attributes ,error: %s\n", fullPath.c_str(), strerror(errno));
                break;
            }
            ///
            /// Attributes of variable.
            ///
            /// #define EFI_VARIABLE_NON_VOLATILE        0x00000001
            /// #define EFI_VARIABLE_BOOTSERVICE_ACCESS  0x00000002
            /// #define EFI_VARIABLE_RUNTIME_ACCESS      0x00000004
            if ((uefiAttributes[0] & 0x01) == 0)
            {
                close(fd);
                delete[] buffer;
                return -1;
            }
        }

        // remove immutable flag
        rc = ioctl(fd, FS_IOC_GETFLAGS, &oflags);
        if (rc < 0)
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: failed to get UEFI var %s flags, error: %s\n", fullPath.c_str(), strerror(errno));
            break;
        }

        // remove immutable flag if needed
        if (oflags & FS_IMMUTABLE_FL)
        {
            oflags &= ~FS_IMMUTABLE_FL;
            rc = ioctl(fd, FS_IOC_SETFLAGS, &oflags);
            if (rc < 0)
            {
                uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: failed to set UEFI var %s flags, error: %s\n", fullPath.c_str(), strerror(errno));
                break;
            }
        }

        // close fd and re-open with write only
        close(fd);
        fd = open(fullPath.c_str(), O_WRONLY);
        if (fd == -1)
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: failed to re-open uefi variable %s, error: %s\n", fullPath.c_str(), strerror(errno));
            break;
        }

        bytesWrote = write(fd, buffer, dataSize + sizeof(attributes));
        if (bytesWrote != (ssize_t)(dataSize + sizeof(attributes)))
        {
            uefi_log_message(MP_REG_LOG_LEVEL_ERROR, "writeUEFIVar: failed to write uefi variable %s, wrote %zu bytes, error: %s\n", fullPath.c_str(), bytesWrote, strerror(errno));
            break;
        }

        bytesWrote -= sizeof(attributes);
    } while (0);

    if (buffer)
    {
        delete[] buffer;
    }
    if (fd != -1)
    {
        close(fd);
    }
    return (int)bytesWrote;
}