package ingress

import (
	"context"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func copySecretsFromIngress(client client.Client, ctx context.Context, ingress *networkingv1.Ingress, sourceNamespace, targetNamespace string) []string {
	var createdSecrets []string

	for _, ingressTLS := range ingress.Spec.TLS {
		log.Debugf("Found usage of Secret [%s] for Hosts %s", ingressTLS.SecretName, ingressTLS.Hosts)

		targetSecretName := types.NamespacedName{
			Namespace: targetNamespace,
			Name:      ingressTLS.SecretName,
		}
		targetSecret := &corev1.Secret{}

		// Check if we need to create the target Secret
		err := client.Get(ctx, targetSecretName, targetSecret)
		if !errors.IsNotFound(err) {
			log.Debugf("Skipping creation of the target Secret [%s] as it already exists", targetSecretName)
			continue
		}

		// Fetch the source Secret
		sourceSecretName := types.NamespacedName{
			Namespace: sourceNamespace,
			Name:      ingressTLS.SecretName,
		}
		sourceSecret := &corev1.Secret{}

		err = client.Get(ctx, sourceSecretName, sourceSecret)
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

		err = client.Create(ctx, targetSecret)
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

	return createdSecrets
}
