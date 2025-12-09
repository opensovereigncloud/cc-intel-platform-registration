package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/config"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/constants"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/metrics"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/registration"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

// Version information
var (
	appName   = "cc-intel-platform-registration"
	version   = "dev"
	buildDate = "unknown"
)

func recoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("panic recovered in HTTP handler", zap.Any("panic", r))
					metrics.IncrementPanicCounts()
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte("Internal Server Error"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// GetRegistrationServicePort retrieves the Regustration service port from environment variables, defaulting to 8080.
func GetRegistrationServicePort(logger *zap.Logger) string {
	port := constants.DefaultRegistrationServicePort
	portStr := os.Getenv(constants.RegistrationServicePortEnv)
	if portStr != "" {
		parsedPort, err := strconv.Atoi(portStr)
		if err != nil {
			logger.Error("failed to parse registration service port",
				zap.String("env_var", constants.RegistrationServicePortEnv),
				zap.Error(err),
				zap.Int("default_value", constants.DefaultRegistrationServicePort))
			// Continue with default value
		} else {
			port = parsedPort
		}
	} else {
		logger.Info("registration service port not set, using default",
			zap.String("env_var", constants.RegistrationServicePortEnv),
			zap.String("default_value", strconv.Itoa(constants.DefaultRegistrationServicePort)))
	}
	// Prepend ":" to form a valid address for http.Server.
	return ":" + strconv.Itoa(port)
}

// GetRegistrationServiceIntervalDuration retrieves the registration service interval from environment variables
func GetRegistrationServiceIntervalDuration(logger *zap.Logger) time.Duration {
	intervalStr := os.Getenv(constants.DefaultRegistrationServiceIntervalInMinutesEnv)
	interval := constants.DefaultRegistrationServiceIntervalInMinutes
	if intervalStr != "" {
		parsedInterval, err := strconv.Atoi(intervalStr)
		if err != nil {
			logger.Error("failed to parse registration service interval",
				zap.String("env_var", constants.DefaultRegistrationServiceIntervalInMinutesEnv),
				zap.Error(err),
				zap.Int("default_value", constants.DefaultRegistrationServiceIntervalInMinutes))
			// Continue with default value
		} else {
			interval = parsedInterval
		}
	} else {
		logger.Info("Registration service interval not set, using default",
			zap.String("env_var", constants.DefaultRegistrationServiceIntervalInMinutesEnv),
			zap.Int("default_value", constants.DefaultRegistrationServiceIntervalInMinutes))
	}
	return time.Duration(interval) * time.Minute
}

// createLogger creates a new zap.Logger with the specified configuration
func createLogger(level string, encoder string, timeEncoding string) (*zap.Logger, error) {
	// Set defaults if not specified
	if level == "" {
		level = "info"
	}

	// Parse the level
	zapLevel, err := zap.ParseAtomicLevel(level)

	if err != nil {
		// Default to info on error
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.CallerKey = ""     // omit the caller
	encoderConfig.StacktraceKey = "" // omit the stack trace

	// Override time encoder if specified
	switch timeEncoding {
	case "rfc3339":
		encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	case "rfc3339nano":
		encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	case "iso8601":
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	case "millis":
		encoderConfig.EncodeTime = zapcore.EpochMillisTimeEncoder
	case "nanos":
		encoderConfig.EncodeTime = zapcore.EpochNanosTimeEncoder
	default:
		encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	}

	cfg := zap.NewProductionConfig()
	cfg.DisableCaller = true
	cfg.EncoderConfig = encoderConfig
	cfg.Level = zapLevel

	if encoder == "json" {
		cfg.Encoding = "json"
	} else {
		cfg.Encoding = "console"
	}

	// Build the logger
	return cfg.Build()
}

// runService starts the registration service and HTTP server
func runService(ctx context.Context, logger *zap.Logger) error {
	// Log application startup information
	logger.Info("Application starting",
		zap.String("app", appName),
		zap.String("version", version),
		zap.String("buildDate", buildDate))

	// Load configuration
	cfg, err := config.LoadRegistrationServiceConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Log configuration (without sensitive data)
	logger.Info("Configuration loaded",
		zap.Int("pccsURLCount", len(cfg.PCCSURLs)),
		zap.Bool("customCACert", cfg.PCCSCACertPath != ""),
		zap.Duration("registrationInterval", cfg.RegistrationInterval),
		zap.Int("servicePort", cfg.ServicePort))

	signalCtx, signalCancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer signalCancel()

	registrationService := registration.NewRegistrationService(logger, cfg, cfg.RegistrationInterval)

	// Create a context with cancel function for shutdown
	g, gCtx := errgroup.WithContext(signalCtx)

	// Start the registration service
	g.Go(func() error {
		// Add panic recovery with metrics
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic in registration service", zap.Any("panic", r))
				metrics.IncrementPanicCounts()
				logger.Info("Incremented the application Panic count metric")

			}
		}()

		return registrationService.Run(gCtx)
	})

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Service is healthy")
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Service is ready")
	})

	// Create server with timeout configuration
	server := &http.Server{
		Addr:    GetRegistrationServicePort(logger),
		Handler: recoveryMiddleware(logger)(mux),
	}

	// Start the HTTP server in a goroutine
	g.Go(func() error {
		logger.Info("Starting HTTP server",
			zap.String("address", server.Addr),
			zap.Int("intervalMinutes", int(cfg.RegistrationInterval.Minutes())))

		serverErr := server.ListenAndServe()
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			logger.Error("http server failed", zap.Error(serverErr))
			return serverErr
		}
		return nil
	})

	// Add server shutdown handler
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("Shutting down HTTP server")

		// Create a timeout context for shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()

		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("http server shutdown error", zap.Error(shutdownErr))
			return shutdownErr
		}

		logger.Info("HTTP server shutdown completed")
		return nil
	})

	// Wait for all goroutines to complete
	waitErr := g.Wait()
	if waitErr != nil && !errors.Is(waitErr, context.Canceled) {
		logger.Error("service error", zap.Error(waitErr))
		return waitErr
	}

	logger.Info("Service shutdown complete")
	return nil
}

func main() {
	// Define command line flags using pflag
	logLevel := pflag.String("zap-log-level", "", "Log level (debug, info, warn, error)")
	encoder := pflag.String("zap-encoder", "json", "Log encoder (json, console)")
	timeEncoding := pflag.String("zap-time-encoding", "rfc3339nano", "Time encoding (rfc3339, rfc3339nano, iso8601, millis, nanos)")

	// Add help flag
	help := pflag.BoolP("help", "h", false, "Display help information")

	// Parse flags
	pflag.Parse()

	// Display help if requested
	if *help {
		fmt.Printf("Usage of %s:\n", appName)
		pflag.PrintDefaults()
		os.Exit(0)
	}

	// Setup panic handler
	defer func() {
		if r := recover(); r != nil {
			// Create a basic logger for panic case
			logger, _ := createLogger("error", "json", "rfc3339nano")
			if logger != nil {
				logger.Error("application panic", zap.Any("panic", r))
			}
			os.Exit(1)
		}
	}()

	// Create logger
	logger, err := createLogger(*logLevel, *encoder, *timeEncoding)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Create context
	ctx := context.Background()

	// Run the service
	err = runService(ctx, logger)

	if err != nil {
		os.Exit(1)
	}

}
