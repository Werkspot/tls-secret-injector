package ingress

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type mutator struct {
	client          client.Client
	sourceNamespace string

	decoder *admission.Decoder
}

func newMutator(client client.Client, sourceNamespace string) *mutator {
	return &mutator{
		client:          client,
		sourceNamespace: sourceNamespace,
	}
}

func (m *mutator) Handle(ctx context.Context, request admission.Request) admission.Response {
	log.Debugf("Received request to mutate Ingress [%s/%s]", request.Namespace, request.Name)

	// Check if the request is the same as the source
	if request.Namespace == m.sourceNamespace {
		reason := fmt.Sprintf("Skipping mutation of Ingress [%s/%s] from the same namespace as the source", request.Namespace, request.Name)
		log.Debug(reason)
		return admission.Allowed(reason)
	}

	// Decode the Ingress from the request
	ingress := &networkingv1.Ingress{}

	err := m.decoder.Decode(request, ingress)
	if err != nil {
		err = fmt.Errorf("failed to decode Ingress [%s/%s]: %v", request.Namespace, request.Namespace, err)
		log.Error(err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Create new Secrets by copying Secrets from the source namespace
	createdSecrets := copySecretsFromIngress(m.client, ctx, ingress, m.sourceNamespace, request.Namespace)

	if len(createdSecrets) == 0 {
		return admission.Allowed("No new Secrets created")
	}

	return admission.Allowed(fmt.Sprintf("Successfully created Secrets %s", createdSecrets))
}

func (m *mutator) InjectDecoder(decoder *admission.Decoder) error {
	m.decoder = decoder
	return nil
}
