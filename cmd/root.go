package cmd

import (
	"fmt"
	"os"
	"strings"

	"tls-secret-injector/pkg/ingress"
	"tls-secret-injector/pkg/secret"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Config holds the application configuration
type Config struct{}

// TLSSecretInjector main application
type TLSSecretInjector struct {
	command *cobra.Command
}

// NewTLSSecretInjector returns a pointer to TLSSecretInjector
func NewTLSSecretInjector() *TLSSecretInjector {
	return &TLSSecretInjector{
		command: getCommand(),
	}
}

// Run the main application
func (app *TLSSecretInjector) Run() int {
	app.command.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return app.initLogger()
	}

	if err := app.command.Execute(); err != nil {
		log.Error(err)
		return 1
	}

	return 0
}

func (app *TLSSecretInjector) initLogger() (err error) {
	level, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		return
	}

	log.SetOutput(os.Stdout)
	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
		ForceColors:            true,
	})

	return
}

func bindFlags(flag *pflag.Flag) {
	viper.RegisterAlias(strings.ReplaceAll(flag.Name, "-", "_"), flag.Name)
}

func getCommand() (c *cobra.Command) {
	pflag.String("cert-dir", "", "Directory that holds the tls.crt and tls.key files")
	pflag.String("leader-election-resource", "", "Resource name that the leader election will use for holding the leader lock")
	pflag.String("leader-election-namespace", "", "Namespace in which the leader election resource will be created")
	pflag.String("log-level", "warning", "Log verbosity level")
	pflag.String("source-namespace", "", "Namespace containing the original TLS Secret from which we want to copy")
	pflag.Parse()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		panic(err)
	}

	pflag.VisitAll(bindFlags)

	return &cobra.Command{
		Use:   "tls-secret-injector",
		Short: "Listen for Ingresses object created and patch them to have a valid certificate",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Setup the manager
			mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
				Host:    "",
				Port:    8443,
				CertDir: viper.GetString("cert-dir"),

				HealthProbeBindAddress: ":8080",
				MetricsBindAddress:     ":8081",

				LeaderElection:             true,
				LeaderElectionID:           viper.GetString("leader-election-resource"),
				LeaderElectionNamespace:    viper.GetString("leader-election-namespace"),
				LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
			})
			if err != nil {
				err = fmt.Errorf("unable to set up overall controller manager: %v", err)
				return
			}

			// Add healthz and readyz check
			err = mgr.AddHealthzCheck("ping", healthz.Ping)
			if err != nil {
				err = fmt.Errorf("failed to add ping healthz check")
				return
			}

			err = mgr.AddReadyzCheck("ping", healthz.Ping)
			if err != nil {
				err = fmt.Errorf("failed to add ping readyz check")
				return
			}

			// Setup a new controller to reconcile Ingresses
			err = ingress.NewController(mgr, viper.GetString("source-namespace"))
			if err != nil {
				return
			}

			// Setup a new controller to reconcile Secrets
			err = secret.NewController(mgr, viper.GetString("source-namespace"))
			if err != nil {
				return
			}

			// Start the controller manager
			log.Infof("Starting controller manager")

			err = mgr.Start(signals.SetupSignalHandler())
			if err != nil {
				err = fmt.Errorf("unable to start manager: %v", err)
				return
			}

			return
		},
	}
}
