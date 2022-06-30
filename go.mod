module github.com/jefedavis/contour-httpproxy-shim

go 1.16

require (
	github.com/cert-manager/cert-manager v1.8.2
	github.com/go-logr/logr v1.2.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/projectcontour/contour v1.21.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/cli-runtime v0.23.4
	k8s.io/client-go v0.23.4
	k8s.io/component-base v0.23.4
	k8s.io/klog/v2 v2.30.0
	sigs.k8s.io/controller-runtime v0.11.1
)
