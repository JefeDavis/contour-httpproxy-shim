package app

import (
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/go-logr/logr"
	"github.com/jefedavis/contour-httpproxy-shim/internal/app/options"
	projectcontourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func setupEventRecorder(s *runtime.Scheme, cl *kubernetes.Clientset, opts *options.Options) logr.Logger {
	logger := opts.Logr.WithName("controller-manager")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(func(format string, args ...interface{}) {
		logger.V(3).Info(fmt.Sprintf(format, args...))
	})

	eventBroadcaster.StartRecordingToSink(&clientcorev1.EventSinkImpl{Interface: cl.CoreV1().Events("")})

	opts.Eventrecorder = eventBroadcaster.NewRecorder(s, corev1.EventSource{Component: "cert-manager-contour-httpproxy"})

	return logger
}

func setupHealthEndpoints(mgr ctrl.Manager) error {
	if err := mgr.AddReadyzCheck("informers_synced", readyzCheck(mgr)); err != nil {
		return fmt.Errorf("unable to set up readiness check, %w", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check, %w", err)
	}

	return nil
}

func setupScheme() *runtime.Scheme {
	s := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(projectcontourv1.AddToScheme(s))
	utilruntime.Must(certmanagerv1.AddToScheme(s))

	//+kubebuilder:scaffold:scheme

	return s
}
