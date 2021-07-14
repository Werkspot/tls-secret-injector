package ingress

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	var createdSecrets []string

	for _, ingressTLS := range ingress.Spec.TLS {
		log.Debugf("Found usage of Secret [%s] for Hosts %s", ingressTLS.SecretName, ingressTLS.Hosts)

		targetSecretName := types.NamespacedName{
			Namespace: request.Namespace,
			Name:      ingressTLS.SecretName,
		}
		targetSecret := &corev1.Secret{}

		// Check if we need to create the target Secret
		err := m.client.Get(ctx, targetSecretName, targetSecret)
		if !errors.IsNotFound(err) {
			log.Debugf("Skipping creation of the target Secret [%s] as it already exists", targetSecretName)
			continue
		}

		// Fetch the source Secret
		sourceSecretName := types.NamespacedName{
			Namespace: m.sourceNamespace,
			Name:      ingressTLS.SecretName,
		}
		sourceSecret := &corev1.Secret{}

		err = m.client.Get(ctx, sourceSecretName, sourceSecret)
		if err != nil {
			log.Errorf("could not fetch the source Secret [%s]: %v", sourceSecretName, err)
			continue
		}

		// Copy Secret data from source to target
		targetSecret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Version,
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: targetSecretName.Namespace,
				Name:      targetSecretName.Name,
				Labels: map[string]string{
					"app.kubernetes.io/name":          "tls-secret-injector",
					"tls-secret-injector/source-name": targetSecretName.Name,
				},
			},
			Type: sourceSecret.Type,
			Data: sourceSecret.Data,
		}

		err = m.client.Create(ctx, targetSecret)
		if errors.IsAlreadyExists(err) {
			// While we already check before if the target Secret exists there could be another request being made for
			// another Ingress that uses the same Secret.
			log.Debugf("Skipping creation of the target Secret [%s] as it already exists", targetSecretName)
			continue
		}
		if err != nil {
			log.Errorf("failed to create the target Secret [%s]: %v", targetSecretName, err)
			continue
		}

		createdSecrets = append(createdSecrets, targetSecretName.String())
		log.Infof("Successfully created Secret [%s]", targetSecretName)
	}

	if len(createdSecrets) == 0 {
		return admission.Allowed("No new Secrets created")
	}

	return admission.Allowed(fmt.Sprintf("Successfully created Secrets %s", createdSecrets))
}

func (m *mutator) InjectDecoder(decoder *admission.Decoder) error {
	m.decoder = decoder
	return nil
}
