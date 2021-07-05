package ingress

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func NewController(mgr manager.Manager, sourceNamespace string) error {
	// Setup the webhooks
	server := mgr.GetWebhookServer()
	server.Register("/mutate", &webhook.Admission{
		Handler: newMutator(mgr.GetClient(), sourceNamespace),
	})

	return nil
}
