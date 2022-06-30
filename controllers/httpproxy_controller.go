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

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/go-logr/logr"
	projectcontourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	issuerDefaultKind        = "Issuer"
	clusterIssuerDefaultKind = "ClusterIssuer"
)

// HTTPProxyReconciler reconciles a HttpProxy object
type HTTPProxyReconciler struct {
	client.Client
	Log                logr.Logger
	Scheme             *runtime.Scheme
	DefaultIssuerName  string
	DefaultIssuerKind  string
	DefaultIssuerGroup string
}

//+kubebuilder:rbac:groups=projectcontour.io,resources=httpproxies,verbs=get;list;watch
//+kubebuilder:rbac:groups=projectcontour.io,resources=httpproxies/status,verbs=get
//+kubebuilder:rbac:groups=cert-manager.io, resources=certificates,verb=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=services/status,verbs=ge

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

	certificate := &cmv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cert-manager.io/v1",
			Kind:       "Certificate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hp.Name,
			Namespace: hp.Namespace,
			// OwnerReferences: []metav1.OwnerReferences{*metav1.NewControllerRef(hp, r.Scheme)},
		},
		Spec: cmv1.CertificateSpec{
			DNSNames:   []string{virtualHost.Fqdn},
			SecretName: virtualHost.TLS.SecretName,
			IssuerRef:  issuer,
			Usages:     cmv1.DefaultKeyUsages(),
		},
	}

	if err := ctrl.SetControllerReference(hp, certificate, r.Scheme); err != nil {
		return err
	}

	err := r.Patch(ctx, certificate, client.Apply, &client.PatchOptions{
		Force:        func() *bool { b := true; return &b }(),
		FieldManager: "cert-manager-contour-httpproxy",
	})
	if err != nil {
		return err
	}

	r.Log.Info("Certificate successfully reconciled")

	return nil
}

func (r *HTTPProxyReconciler) getIssuer(hp *projectcontourv1.HTTPProxy) cmmetav1.ObjectReference {
	issuer := cmmetav1.ObjectReference{
		Group: r.DefaultIssuerGroup,
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
	certificate := &cmv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cert-manager.io/v1",
			Kind:       cmv1.CertificateKind,
		},
	}

	return ctrl.NewControllerManagedBy(mgr).For(&projectcontourv1.HTTPProxy{}).Owns(certificate).Complete(r)
}
