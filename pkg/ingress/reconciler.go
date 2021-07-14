package ingress

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type reconciler struct {
	client          client.Client
	sourceNamespace string
}

func newReconciler(client client.Client, sourceNamespace string) *reconciler {
	return &reconciler{
		client:          client,
		sourceNamespace: sourceNamespace,
	}
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, err error) {
	log.Debugf("Received request to reconcile Ingress [%s]", request.NamespacedName)

	// Check if the request is the same as the source
	if request.Namespace == r.sourceNamespace {
		log.Debugf("Skipping mutation of Ingress [%s/%s] from the same namespace as the source", request.Namespace, request.Name)
		return
	}

	// Fetch the Ingress from cache
	ingress := &networkingv1.Ingress{}

	err = r.client.Get(ctx, request.NamespacedName, ingress)
	if errors.IsNotFound(err) {
		log.Debugf("Skipping reconciliation of Ingress [%s] as it no longer exists: %v", request.NamespacedName, err)
		return
	}
	if err != nil {
		err = fmt.Errorf("could not fetch the Ingress [%s]: %v", request.NamespacedName, err)
		log.Error(err)
		return
	}

	// Create new Secrets by copying Secrets from the source namespace
	copySecretsFromIngress(r.client, ctx, ingress, r.sourceNamespace, request.Namespace)

	return
}
