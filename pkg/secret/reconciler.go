package secret

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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
	log.Debugf("Received request to reconcile Secret [%s]", request.NamespacedName)

	if request.Namespace != r.sourceNamespace {
		log.Debugf("Skipping reconciliation of Secret [%s] as it is not from the source namespace", request.NamespacedName)
		return
	}

	// Fetch the source Secret from cache
	sourceSecret := &corev1.Secret{}

	err = r.client.Get(ctx, request.NamespacedName, sourceSecret)
	if errors.IsNotFound(err) {
		log.Debugf("Skipping reconciliation of Secret [%s] as it no longer exists: %v", request.NamespacedName, err)
		return
	}
	if err != nil {
		err = fmt.Errorf("could not fetch the source Secret [%s]: %v", request.NamespacedName, err)
		log.Error(err)
		return
	}

	// Skip if this Secret is not a TLS
	if sourceSecret.Type != corev1.SecretTypeTLS {
		log.Debugf("Skipping reconciliation of Secret [%s] as it is not a TLS Secret", request.NamespacedName)
		return
	}

	// Fetch all Secrets that were created from this Secret
	secretLabels := client.MatchingLabels{
		"app.kubernetes.io/name":          "tls-secret-injector",
		"tls-secret-injector/source-name": request.Name,
	}

	secretMetadataList := &metav1.PartialObjectMetadataList{}
	secretMetadataList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "SecretList",
	})

	err = r.client.List(ctx, secretMetadataList, secretLabels)
	if err != nil {
		err = fmt.Errorf("could not list Secrets: %v", err)
		log.Error(err)
		return
	}

	// Iterate through the list of Secrets metadata
	for _, targetSecretMetadata := range secretMetadataList.Items {
		targetSecretName := types.NamespacedName{
			Namespace: targetSecretMetadata.ObjectMeta.Namespace,
			Name:      targetSecretMetadata.ObjectMeta.Name,
		}

		log.Debugf("Found target Secret [%s] to be copied from source Secret [%s]", targetSecretName, request.NamespacedName)

		// Fetch the target Secret
		targetSecret := &corev1.Secret{}

		err = r.client.Get(ctx, targetSecretName, targetSecret)
		if err != nil {
			err = fmt.Errorf("could not fetch the target Secret [%s]: %v", request.NamespacedName, err)
			log.Error(err)
			return
		}

		// Copy Secret data from source to target
		targetSecret.Data = sourceSecret.Data

		err = r.client.Update(ctx, targetSecret)
		if err != nil {
			err = fmt.Errorf("failed to update target Secret [%s]: %v", targetSecretName, err)
			log.Error(err)
			return
		}

		log.Infof("Successfully updated Secret [%s]", targetSecretName)
	}

	return
}
