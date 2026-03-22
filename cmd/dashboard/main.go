package main

import (
	"flag"
	"os"

	operatorrepo "github.com/Iam-Karan-Suresh/operator-repo"
	"github.com/Iam-Karan-Suresh/operator-repo/api/v1"
	"github.com/Iam-Karan-Suresh/operator-repo/internal/dashboard"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
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

	// Setup K8s Client
	config := ctrl.GetConfigOrDie()
	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to create client")
		os.Exit(1)
	}

	// Setup Server
	server := dashboard.NewServer(k8sClient, port)

	// Extract the embedded filesystem so it can be served
	subFS, err := operatorrepo.GetStaticFS()
	if err != nil {
		setupLog.Error(err, "failed to get sub filesystem for static files")
		os.Exit(1)
	}

	setupLog.Info("Dashboard starting standalone in container", "port", port)

	// Create a new context that we can cancel on SIGTERM
	ctx := ctrl.SetupSignalHandler()

	go func() {
		// As a standalone binary, we'll let the dashboard server itself just run
		// But we need to inject the static file serving logic into the handler
		// Since we didn't inject the FS into the server struct, we'll just let
		// the server handle the API routes, and we'll override the static handler
		// in `handlers.go` via a quick hack or we can reconstruct the mux here.

		// For robustness, let's just let the server Start() run in a goroutine
		// It creates its own mux. We will need to inject the SubFS into the Server struct if we want it perfect,
		// but since `server.Start()` creates its own Mux internally and blocks, we will update `server.go` to accept the FS.
	}()

	err = server.StartWithFS(ctx, subFS)
	if err != nil {
		setupLog.Error(err, "unable to start dashboard server")
		os.Exit(1)
	}
}
