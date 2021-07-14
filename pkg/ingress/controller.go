package ingress

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func NewController(mgr manager.Manager, sourceNamespace string) error {
	// Setup the webhooks
	server := mgr.GetWebhookServer()
	server.Register("/mutate", &webhook.Admission{
		Handler: newMutator(mgr.GetClient(), sourceNamespace),
	})

	// Setup the reconciler
	ingressController, err := controller.New("ingress", mgr, controller.Options{
		Reconciler: newReconciler(mgr.GetClient(), sourceNamespace),
	})
	if err != nil {
		return fmt.Errorf("unable to set up Ingress controller: %v", err)
	}

	// Watch Ingress and enqueue Ingress object key
	err = ingressController.Watch(
		&source.Kind{
			Type: &networkingv1.Ingress{},
		},
		&handler.EnqueueRequestForObject{},
		predicate.Funcs{
			DeleteFunc: func(event event.DeleteEvent) bool {
				log.Debugf(
					"Skipping reconciliation of Ingress [%s/%s] as it has been deleted",
					event.Object.GetNamespace(),
					event.Object.GetName(),
				)
				return false
			},
			GenericFunc: func(event event.GenericEvent) bool {
				log.Debugf(
					"Skipping reconciliation of Ingress [%s/%s] for the generic event type",
					event.Object.GetNamespace(),
					event.Object.GetName(),
				)
				return false
			},
		},
	)
	if err != nil {
		return fmt.Errorf("unable to watch Ingress: %v", err)
	}

	return nil
}
