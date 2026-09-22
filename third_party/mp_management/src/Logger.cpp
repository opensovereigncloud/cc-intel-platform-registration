#include <cstdio>
#include <cstdarg>

#include "logger.h"

extern "C" void goZapLogBridge(int component, int level, char *msg);

// extern "C" LogLevel glog_level;

namespace
{

    void format_and_send(int component, int level, const char *format, va_list args)
    {
        char buffer[1024];
        vsnprintf(buffer, sizeof(buffer), format, args);
        goZapLogBridge(component, level, buffer);
    }

} // namespace

extern "C" void uefi_log_message(LogLevel level, const char *format, ...)
{
    va_list args;
    va_start(args, format);
    format_and_send(0, static_cast<int>(level), format, args);
    va_end(args);
}

extern "C" void management_log_message(LogLevel level, const char *format, ...)
{
    va_list args;
    va_start(args, format);
    format_and_send(1, static_cast<int>(level), format, args);
    va_end(args);
}