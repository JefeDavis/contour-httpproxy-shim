package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

func readyzCheck(mgr ctrl.Manager) func(*http.Request) error {
	return func(req *http.Request) error {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()

		if mgr.GetCache().WaitForCacheSync(ctx) {
			return nil
		}

		return fmt.Errorf("informers not synced")
	}
}
