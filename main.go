package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/constants"
	"github.com/opensovereigncloud/cc-intel-platform-registration/pkg/registration"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func GetRegistrationServiceIntervalTime(log *slog.Logger) time.Duration {
	intervalStr := os.Getenv(constants.REGISTRATION_SERVICE_INTERVAL_IN_MINUTES_ENV)
	interval := constants.DEFAULT_REGISTRATION_SERVICE_INTERVAL_IN_MINUTES
	if intervalStr != "" {
		parsedInterval, err := strconv.Atoi(intervalStr)
		if err != nil {
			log.Error(
				fmt.Sprintf("Failed to parse %s: %v. Using default value: %d",
					constants.REGISTRATION_SERVICE_INTERVAL_IN_MINUTES_ENV,
					err, constants.DEFAULT_REGISTRATION_SERVICE_INTERVAL_IN_MINUTES))
		} else {
			interval = parsedInterval
		}
	} else {
		log.Info(fmt.Sprintf("%s not set. Using default value: %d",
			constants.REGISTRATION_SERVICE_INTERVAL_IN_MINUTES_ENV,
			constants.DEFAULT_REGISTRATION_SERVICE_INTERVAL_IN_MINUTES))
	}
	return time.Duration(interval)
}

// createLogger creates a new slog.Logger with the specified level
func createLogger(level zap.AtomicLevel) *slog.Logger {

	zc := zap.NewProductionConfig()
	zc.Level = level
	z, err := zc.Build()
	if err != nil {
		panic("Unable to create Logger")
	}
	zapLog := zapr.NewLogger(z)
	return slog.New(logr.ToSlogHandler(zapLog))

}

type contextKey string

const LOGGER_CONTEXT_KEY contextKey = "logger"

// setLogConfig sets the log config.
func setLogConfig(cmd *cobra.Command, _ []string) *slog.Logger {
	// Default to Info level
	logLevel := zap.NewAtomicLevel()

	// Try to get level from flag
	logLevelFlag, err := cmd.Flags().GetString("log-level")
	if err != nil {
		slog.Error("internal error: unable to get log-level flag",
			"error", err)
	}

	if logLevelFlag != "" {
		if level, err := zap.ParseAtomicLevel(logLevelFlag); err == nil {
			logLevel = level
		} else {
			slog.Error("unable to set log level using flag",
				"flag", "log-level",
				"value", logLevelFlag,
				"error", err)
		}
	} else {
		// Try to get level from environment variable
		logLevelEnv := os.Getenv(constants.LOG_LEVEL_ENV)
		if logLevelEnv != "" {
			if level, err := zap.ParseAtomicLevel(logLevelEnv); err == nil {
				logLevel = level
			} else {
				slog.Error("unable to set log level from environment variable",
					"env", constants.LOG_LEVEL_ENV,
					"value", logLevelEnv,
					"error", err)
			}
		}
	}

	// Create and return the configured logger
	logger := createLogger(logLevel)

	// Log the configured level
	logger.Debug("log level configured",
		"level", logLevel.String())

	return logger
}

func StartCmdFunc(cmd *cobra.Command, args []string) {
	logger := cmd.Context().Value(LOGGER_CONTEXT_KEY).(*slog.Logger)

	intervalDuration := GetRegistrationServiceIntervalTime(logger)
	registrationService := registration.NewRegistrationService(logger, intervalDuration)
	ctx, cancelFunc := context.WithCancel(context.TODO())
	g, ctx := errgroup.WithContext(ctx)

	defer cancelFunc()

	g.Go(func() error {
		return registrationService.Run(ctx)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Registration Service run start failed: %v", err))
		return
	}
	// Setup HTTP server
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Service is healthy")
	})

	// Start the server
	logger.Debug(fmt.Sprintf("Starting server on :8080 with %d minute interval\n", intervalDuration))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cc-intel-platform-registration",
	Short: "Manage intel platform registrations on k8s clusters",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger := setLogConfig(cmd, args)

		ctx := context.WithValue(cmd.Context(), LOGGER_CONTEXT_KEY, logger)
		cmd.SetContext(ctx)

	},
}

func buildStartCmd() *cobra.Command {
	var command = &cobra.Command{
		Use:   "start",
		Short: "Starts the daemon service",
		Run:   StartCmdFunc,
	}
	return command
}

func main() {
	// Add global flags
	rootCmd.PersistentFlags().String(constants.LOG_LEVEL_FLAG, "", "log level")

	// Add subcommands
	rootCmd.AddCommand(buildStartCmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
