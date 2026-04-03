// Package main provides a standalone entry point for the EC2 Operator Dashboard.
// This allows running the dashboard as a separate container/process from the main controller
// if needed, which is useful for specialized deployment scenarios (like read-only views).
package main

import (
	"flag"
	"os"

	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	operatorrepo "github.com/Iam-Karan-Suresh/operator-repo"
	computev1 "github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/Iam-Karan-Suresh/operator-repo/internal/dashboard"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(computev1.AddToScheme(scheme))
}

func main() {
	var port string
	flag.StringVar(&port, "port", ":3000", "The address the dashboard endpoint binds to.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Setup K8s Client using the default In-Cluster or Kubeconfig context.
	config := ctrl.GetConfigOrDie()
	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to create client")
		os.Exit(1)
	}

	// Setup Server and CostService dependencies.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		setupLog.Error(err, "unable to create kubernetes clientset")
		os.Exit(1)
	}

	dashServer := dashboard.NewServer(k8sClient, clientset, port) // Changed mgr.GetClient() to k8sClient and port variable

	// Extract the embedded filesystem (the React build) so it can be served via HTTP.
	subFS, err := operatorrepo.GetStaticFS()
	if err != nil {
		setupLog.Error(err, "failed to get sub filesystem for static files")
		os.Exit(1)
	}

	setupLog.Info("Dashboard starting standalone in container", "port", port)

	// Create a new context that we can cancel on SIGTERM
	ctx := ctrl.SetupSignalHandler()

	go func() {
		// Run the dashboard cost service sync routine
		dashServer.GetCostService().StartSync(ctx)

		// As a standalone binary, we'll let the dashboard server itself just run
		// But we need to inject the static file serving logic into the handler
		// Since we didn't inject the FS into the server struct, we'll just let
		// the server handle the API routes, and we'll override the static handler
		// in `handlers.go` via a quick hack or we can reconstruct the mux here.

		// For robustness, let's just let the server Start() run in a goroutine
		// It creates its own mux. We will need to inject the SubFS into the Server struct if we want it perfect,
		// but since `server.Start()` creates its own Mux internally and blocks, we will update `server.go` to accept the FS.
	}()

	err = dashServer.StartWithFS(ctx, subFS)
	if err != nil {
		setupLog.Error(err, "unable to start dashboard server")
		os.Exit(1)
	}
}
