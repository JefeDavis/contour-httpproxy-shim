package options

import (
	"flag"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
)

// Options is the main configuration struct for cert-manager-contour-httpproxies.
type Options struct {
	Eventrecorder record.EventRecorder

	// DefaultIssuerName is the name of the default issuer to use.
	DefaultIssuerName string

	// DefaultIssuerKind is the default issuer kind to use.
	DefaultIssuerKind string

	// DefaultIssuerGroup is the default issuer group to use.
	DefaultIssuerGroup string

	// ReadyzPort is the port to serve the readiness check on.
	ReadyzPort int

	// ReadyzPath is the path to serve the readiness check on.
	ReadyzPath string

	// MetricsPort is the port to serve prometheus metrics on.
	MetricsPort int

	// EnableLeaderElection determines whether to use leader election.
	EnableLeaderElection bool

	// LeaderElectionNamespace is the namespace to create Leader Election Resources.
	LeaderElectionNamespace string

	// Logr is the shared base logr.Logger
	Logr logr.Logger

	// RestConfig is the Kubernetes Config.
	RestConfig *rest.Config

	logLevel string

	kubeConfigFlags *genericclioptions.ConfigFlags
}

func New() *Options {
	return new(Options)
}

func (o *Options) Prepare(cmd *cobra.Command) {
	o.addFlags(cmd)
}

func (o *Options) Complete() error {
	klog.InitFlags(nil)
	log := klogr.New()

	if err := flag.Set("v", o.logLevel); err != nil {
		return err
	}

	o.Logr = log.WithName("cert-manager-contour-httpproxies")

	var err error
	o.RestConfig, err = o.kubeConfigFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to build kubernetes rest config: %s", err)
	}

	return nil
}

func (o *Options) addFlags(cmd *cobra.Command) {
	var nfs cliflag.NamedFlagSets

	o.addAppFlags(nfs.FlagSet("App"))
	o.kubeConfigFlags = genericclioptions.NewConfigFlags(true)
	o.kubeConfigFlags.AddFlags(nfs.FlagSet("Kubernetes"))

	usageFmt := "Usage:\n %s\n"

	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), nfs, 0)

		return nil
	})

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStderr(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), nfs, 0)
	})

	fs := cmd.Flags()
	for _, f := range nfs.FlagSets {
		fs.AddFlagSet(f)
	}
}

func (o *Options) addAppFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.DefaultIssuerName, "default-issuer-name", "", "Name of the default cert issuer to use")
	fs.StringVar(&o.DefaultIssuerKind, "default-issuer-kind", "Issuer", "default cert issuer kind to use")
	fs.StringVar(&o.DefaultIssuerGroup, "default-issuer-group", "cert-manager.io/v1", "default cert issuer group to use")
	fs.StringVarP(&o.logLevel, "log-level", "v", "1", "Log level (1-5).")
	fs.IntVar(&o.ReadyzPort, "readiness-probe-port", 6060, "Port to expose the readiness probe.")
	fs.StringVar(&o.ReadyzPath, "readiness-probe-path", "/readyz", "HTTP Path to expose the readiness probe server.")
	fs.IntVar(&o.MetricsPort, "metrics-port", 9402, "Port to expose Prometheus metrics on 0.0.0.0d on path '/metrics'.")
	fs.BoolVar(&o.EnableLeaderElection, "enable-leader-election", true, "enable leader election on the controller.")
	fs.StringVar(&o.LeaderElectionNamespace, "leader-election,namespace", "cert-manager", "Namespace to create leader election resources in.")
}
