package main

import (
	"runtime"

	"github.com/prune998/certmerge-operator/pkg/apis"
	"github.com/prune998/certmerge-operator/pkg/controller"

	sdkVersion "github.com/operator-framework/operator-sdk/version"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
)

func printVersion() {
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Printf("operator-sdk Version: %v", sdkVersion.Version)
}

var (
	logLevel = flag.String("loglevel", log.WarnLevel.String(), "the log level to display")
)

func main() {
	flag.Parse()
	printVersion()

	// set logs in json format
	myLogLevel, err := log.ParseLevel(*logLevel)
	if err != nil {
		myLogLevel = log.WarnLevel
	}
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(myLogLevel)

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: ""})
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal(err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Fatal(err)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	log.Fatal(mgr.Start(signals.SetupSignalHandler()))
}
