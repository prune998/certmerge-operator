package main

import (
	"context"
	"os"
	"runtime"

	"github.com/prune998/certmerge-operator/pkg/apis"
	"github.com/prune998/certmerge-operator/pkg/controller"

	"github.com/operator-framework/operator-sdk/pkg/leader"
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
	logLevel       = flag.String("loglevel", log.WarnLevel.String(), "the log level to display")
	logJSON        = flag.Bool("logjson", true, "log to stdlog using JSON format")
	displayVersion = flag.Bool("version", false, "Show version and quit")
)

func main() {
	flag.Parse()

	if *displayVersion {
		printVersion()
		os.Exit(0)
	}

	// set log level and json format
	myLogLevel, err := log.ParseLevel(*logLevel)
	if err != nil {
		myLogLevel = log.WarnLevel
	}
	log.SetLevel(myLogLevel)

	if *logJSON {
		log.SetFormatter(&log.JSONFormatter{})
	}

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

	// Become the leader before proceeding
	err = leader.Become(context.TODO(), "certmerge-operator")
	if err != nil {
		log.Panicf("Error becoming the leader - %v", err)
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
