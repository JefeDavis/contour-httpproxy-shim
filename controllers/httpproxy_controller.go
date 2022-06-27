/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	projectcontourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	issuerAnnotation           = "cert-manager.io/issuer"
	clusterIssuerAnnotation    = "cert-manager.io/cluster-issuer"
	ingressClassNameAnnotation = "kubernetes.io/ingress.class"
	issuerKindAnnotation       = "cert-manager.io/issuer-kind"
	issuerGroupAnnotation      = "cert-manager.io/issuer-group"
)

const (
	certificateKind          = "Certificate"
	issuerDefaultKind        = "Issuer"
	clusterIssuerDefaultKind = "ClusterIssuer"
)

const (
	usageDigitalSignature = "digital signature"
	usageKeyEncipherment  = "key encipherment"
	usageServerAuth       = "server auth"
	usageClientAuth       = "client auth"
)

// HTTPProxyReconciler reconciles a HttpProxy object
type HTTPProxyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type certIssuer struct {
	Group string
	Kind  string
	Name  string
}

//+kubebuilder:rbac:groups=projectcontour.io,resources=httpproxies,verbs=get;list;watch
//+kubebuilder:rbac:groups=projectcontour.io,resources=httpproxies/status,verbs=get
//+kubebuilder:rbac:groups=cert-manager.io, resources=certificates,verb=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=services/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HttpProxy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *HTTPProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = log.FromContext(ctx)

	hp := new(projectcontourv1.HTTPProxy)

	objKey := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}

	err := r.Get(ctx, objKey, hp)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		r.Log.Error(err, "unable to get HTTPProxy resources")
		return ctrl.Result{}, err
	}

	//skip reconcile if already deleting
	if hp.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if err := r.reconcileCertificate(ctx, hp); err != nil {
		r.Log.Error(err, "unable to reconcile Certificate")

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *HTTPProxyReconciler) reconcileCertificate(ctx context.Context, hp *projectcontourv1.HTTPProxy) error {
	virtualHost := hp.Spec.VirtualHost
	switch {
	case virtualHost == nil:
		return nil
	case virtualHost.Fqdn == "":
		return nil
	case virtualHost.TLS == nil:
		return nil
	case virtualHost.TLS.SecretName == "":
		return nil
	}

	issuer := r.getIssuer(hp)

	if issuer.Name == "" {
		r.Log.Info("no issuer found, skipping")
		return nil
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       certificateKind,
			"spec": map[string]interface{}{
				"dnsNames":   []string{virtualHost.Fqdn},
				"secretName": virtualHost.TLS.SecretName,
				"commonName": virtualHost.Fqdn,
				"issuerRef": map[string]interface{}{
					"group": issuer.Group,
					"kind":  issuer.Kind,
					"name":  issuer.Name,
				},
				"usages": []string{
					usageDigitalSignature,
					usageKeyEncipherment,
					usageServerAuth,
					usageClientAuth,
				},
			},
		},
	}

	obj.SetNamespace(hp.Name)
	obj.SetNamespace(hp.Namespace)

	err := ctrl.SetControllerReference(hp, obj, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Patch(ctx, obj, client.Apply, &client.PatchOptions{
		Force:        func() *bool { b := true; return &b }(),
		FieldManager: "cert-manager-httpproxy-shim",
	})
	if err != nil {
		return err
	}

	r.Log.Info("Certificate successfully reconciled")

	return nil
}

func (r *HTTPProxyReconciler) getIssuer(hp *projectcontourv1.HTTPProxy) *certIssuer {
	issuer := &certIssuer{
		Group: "cert-manager.io",
	}

	if name, ok := hp.Annotations[clusterIssuerAnnotation]; ok {
		issuer.Name = name

		if kind, ok := hp.Annotations[clusterIssuerDefaultKind]; ok {
			issuer.Kind = kind
		} else {
			issuer.Kind = clusterIssuerDefaultKind
		}
	}

	if name, ok := hp.Annotations[issuerAnnotation]; ok {
		issuer.Name = name

		if kind, ok := hp.Annotations[issuerKindAnnotation]; ok {
			issuer.Kind = kind

		} else {
			issuer.Kind = issuerDefaultKind
		}
	}

	if group, ok := hp.Annotations[issuerGroupAnnotation]; ok {
		issuer.Group = group
	}

	return issuer
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       certificateKind,
		},
	}

	return ctrl.NewControllerManagedBy(mgr).For(&projectcontourv1.HTTPProxy{}).Owns(obj).Complete(r)

}
