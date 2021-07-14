package secret

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	sourceSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "source",
			Name:      "tls-example-io",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("private key"),
		},
	}

	targetSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "target",
			Name:      "tls-example-io",
			Labels: map[string]string{
				"app.kubernetes.io/name":          "tls-secret-injector",
				"tls-secret-injector/source-name": sourceSecret.ObjectMeta.Name,
			},
			ResourceVersion: "1",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("outdated certificate"),
			corev1.TLSPrivateKeyKey: []byte("outdated private key"),
		},
	}

	sourceSecretName := types.NamespacedName{
		Namespace: sourceSecret.ObjectMeta.Namespace,
		Name:      sourceSecret.ObjectMeta.Name,
	}

	targetSecretName := types.NamespacedName{
		Namespace: targetSecret.ObjectMeta.Namespace,
		Name:      targetSecret.ObjectMeta.Name,
	}

	request := reconcile.Request{
		NamespacedName: sourceSecretName,
	}

	// Create a client and the reconciler
	fakeClient := fake.NewClientBuilder().WithObjects(sourceSecret, targetSecret).Build()
	reconciler := newReconciler(fakeClient, sourceSecret.ObjectMeta.Namespace)

	// Reconcile and check for errors
	_, err := reconciler.Reconcile(context.TODO(), request)
	assert.NoError(t, err)

	// Retrieve and verify if the Secret was updated
	updatedSecret := &corev1.Secret{}
	err = fakeClient.Get(context.TODO(), targetSecretName, updatedSecret)

	assert.NoError(t, err)
	assert.Equal(t, "2", updatedSecret.ResourceVersion)
	assert.Equal(t, "certificate", string(updatedSecret.Data[corev1.TLSCertKey]))
	assert.Equal(t, "private key", string(updatedSecret.Data[corev1.TLSPrivateKeyKey]))
}
