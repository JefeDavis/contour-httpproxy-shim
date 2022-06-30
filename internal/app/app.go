package app

import (
	"fmt"

	"github.com/jefedavis/contour-httpproxy-shim/controllers"
	"github.com/jefedavis/contour-httpproxy-shim/internal/app/options"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Command is the main entrypoint into the controler setup.
func Command() *cobra.Command {
	opts := options.New()
	cmd := &cobra.Command{
		Use:   "cert-manager-contour-httpproxy",
		Short: "cert-manager support for contour httpproxies",
		Long:  "cert-manager support for contour httpproxies",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(); err != nil {
				return err
			}

			cl, err := kubernetes.NewForConfig(opts.RestConfig)
			if err != nil {
				return fmt.Errorf("error creating kubernetes client: %w", err)
			}

			if err := validateCustomResourceExists("projectcontour.io/v1", "HTTPProxy", cl); err != nil {
				return err
			}

			if err := validateCustomResourceExists("cert-manager.io/v1", "CertificateRequest", cl); err != nil {
				return err
			}

			scheme := setupScheme()

			logger := setupEventRecorder(scheme, cl, opts)

			mgr, err := ctrl.NewManager(opts.RestConfig, ctrl.Options{
				Scheme:                        scheme,
				Logger:                        logger,
				LeaderElection:                opts.EnableLeaderElection,
				LeaderElectionID:              "cert-manager-contour-httpproxy",
				LeaderElectionNamespace:       opts.LeaderElectionNamespace,
				LeaderElectionResourceLock:    "leases",
				LeaderElectionReleaseOnCancel: true,
				ReadinessEndpointName:         opts.ReadyzPath,
				HealthProbeBindAddress:        fmt.Sprintf("[::]:%d", opts.ReadyzPort),
				MetricsBindAddress:            fmt.Sprintf("[::]:%d", opts.MetricsPort),
			})
			if err != nil {
				return fmt.Errorf("could not create controller manager, %w", err)
			}

			if err := controllers.SetupReconciler(mgr, scheme, controllers.ReconcilerOptions{
				DefaultIssuerName:  opts.DefaultIssuerName,
				DefaultIssuerKind:  opts.DefaultIssuerKind,
				DefaultIssuerGroup: opts.DefaultIssuerGroup,
			}); err != nil {
				return fmt.Errorf("could not create controller, %w", err)
			}

			opts.Logr.V(5).Info("starting controller")
			return mgr.Start(ctrl.SetupSignalHandler())
		},
	}

	opts.Prepare(cmd)

	return cmd
}

// validateHTTPProxiesExist checks if v1 contour HTTPProxies exist in the API server
func validateCustomResourceExists(group, kind string, cl *kubernetes.Clientset) error {

	resources, err := cl.Discovery().ServerResourcesForGroupVersion(group)
	if err != nil {
		return fmt.Errorf("couldn't check if %s exists in the kubernetes API, %w", group, err)
	}

	for _, r := range resources.APIResources {
		if r.Kind == kind {
			return nil
		}
	}

	return fmt.Errorf("connected to the Kuberentes API, but the %s %s CRD does not appear to be installed", group, kind)
}
