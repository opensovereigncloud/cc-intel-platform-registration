package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"cc-intel-platform-registration/pkg/constants"
	"cc-intel-platform-registration/pkg/registration"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func GetRegistrationServiceIntervalTime() time.Duration {
	intervalStr := os.Getenv(constants.REGISTRATION_SERVICE_INTERVAL_IN_MINUTES_ENV)
	interval := constants.DEFAULT_REGISTRATION_SERVICE_INTERVAL_IN_MINUTES
	if intervalStr != "" {
		parsedInterval, err := strconv.Atoi(intervalStr)
		if err != nil {
			log.Fatalf("Failed to parse %s: %v. Using default value: %d",
				constants.REGISTRATION_SERVICE_INTERVAL_IN_MINUTES_ENV,
				err, constants.DEFAULT_REGISTRATION_SERVICE_INTERVAL_IN_MINUTES)
		} else {
			interval = parsedInterval
		}
	} else {
		log.Infof("%s not set. Using default value: %d",
			constants.REGISTRATION_SERVICE_INTERVAL_IN_MINUTES_ENV,
			constants.DEFAULT_REGISTRATION_SERVICE_INTERVAL_IN_MINUTES)
	}
	return time.Duration(interval)
}

// setLogConfig sets the log config.
func setLogConfig(cmd *cobra.Command, args []string) {
	logLevelFlag, err := cmd.Flags().GetString(constants.LOG_LEVEL_FLAG)
	if err != nil {
		log.Fatalf("internal error: unable to get %s flag", constants.LOG_LEVEL_FLAG)
	}

	if logLevelFlag != "" {
		if logLevel, err := log.ParseLevel(logLevelFlag); err == nil {
			log.SetLevel(logLevel)
		} else {
			log.Fatalf("unable to set log level using %s flag: %s", constants.LOG_LEVEL_FLAG, err)
		}
	} else {
		logLevelEnv := os.Getenv(constants.LOG_LEVEL_ENV)
		if logLevelEnv != "" {
			if logLevel, err := log.ParseLevel(logLevelEnv); err == nil {
				log.SetLevel(logLevel)
			} else {
				log.Fatalf("unable to set log level from %s env: %s", constants.LOG_LEVEL_ENV, err)
			}
		}
	}
	log.Debug("log level set to ", log.GetLevel())
}

func StartCmdFunc(cmd *cobra.Command, args []string) {
	// logrus.SetLevel(logrus.DebugLevel)
	intervalDuration := GetRegistrationServiceIntervalTime()
	registrationService := registration.NewRegistrationService(intervalDuration)
	ctx, cancelFunc := context.WithCancel(context.TODO())
	defer cancelFunc()

	go registrationService.Run(ctx)

	// Setup HTTP server
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Service is healthy")
	})

	// Start the server
	// log.Printf("Starting server on :8080 with %d minute interval\n", interval)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:              "cc-intel-platform-registration",
	Short:            "Manage intel platform registrations on k8s clusters",
	PersistentPreRun: setLogConfig,
}

func buildStartCmd() *cobra.Command {
	var command = &cobra.Command{
		Use:   "start",
		Short: "Starts the daemon service",
		Run:   StartCmdFunc,
	}
	return command
}

func RunCli() {
	// Add global flags
	rootCmd.PersistentFlags().String(constants.LOG_LEVEL_FLAG, "", "log level")

	// Add subcommands
	rootCmd.AddCommand(buildStartCmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
