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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ReconcilerOptions contains options for the reconciler.
type ReconcilerOptions struct {
	DefaultIssuerName  string
	DefaultIssuerKind  string
	DefaultIssuerGroup string
}

// SetupReconciler initializes the reconciler.
func SetupReconciler(mgr manager.Manager, scheme *runtime.Scheme, opts ReconcilerOptions) error {
	httpProxyReconciler := &HTTPProxyReconciler{
		Client:             mgr.GetClient(),
		Log:                ctrl.Log.WithName("controllers").WithName("cert-manager-contour-httpproxy"),
		Scheme:             scheme,
		DefaultIssuerName:  opts.DefaultIssuerName,
		DefaultIssuerKind:  opts.DefaultIssuerKind,
		DefaultIssuerGroup: opts.DefaultIssuerGroup,
	}

	if err := httpProxyReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	// +kubebuilder:scaffold:builder
	return nil
}
