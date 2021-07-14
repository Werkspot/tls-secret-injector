package secret

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func NewController(mgr manager.Manager, sourceNamespace string) error {
	// Setup the reconciler
	secretController, err := controller.New("secret", mgr, controller.Options{
		Reconciler: newReconciler(mgr.GetClient(), sourceNamespace),
	})
	if err != nil {
		return fmt.Errorf("unable to set up Secret controller: %v", err)
	}

	// Watch Secret and enqueue Secret object key
	err = secretController.Watch(
		&source.Kind{
			Type: &corev1.Secret{},
		},
		&handler.EnqueueRequestForObject{},
		predicate.Funcs{
			CreateFunc: func(event event.CreateEvent) bool {
				log.Debugf(
					"Skipping reconciliation of Secret [%s/%s] as it has been created",
					event.Object.GetNamespace(),
					event.Object.GetName(),
				)
				return false
			},
			DeleteFunc: func(event event.DeleteEvent) bool {
				log.Debugf(
					"Skipping reconciliation of Secret [%s/%s] as it has been deleted",
					event.Object.GetNamespace(),
					event.Object.GetName(),
				)
				return false
			},
			GenericFunc: func(event event.GenericEvent) bool {
				log.Debugf(
					"Skipping reconciliation of Secret [%s/%s] for the generic event type",
					event.Object.GetNamespace(),
					event.Object.GetName(),
				)
				return false
			},
		},
	)
	if err != nil {
		return fmt.Errorf("unable to watch Secret: %v", err)
	}

	return nil
}
