package ingress

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	tests := map[string]struct {
		ingress   networkingv1.Ingress
		objects   []client.Object
		newSecret corev1.Secret
		reason    string
	}{
		"skip when same namespace": {
			ingress: *newIngress("source"),
			reason:  "Skipping mutation of Ingress [source/example-io] from the same namespace as the source",
		},
		"create target secret": {
			ingress: *newIngress("target"),
			objects: []client.Object{
				newSecret("source"),
			},
			newSecret: *newSecret("target"),
			reason:    "Successfully created Secrets [target/tls-example-io]",
		},
		"skip creation of target secret": {
			ingress: *newIngress("target"),
			objects: []client.Object{
				newSecret("source"),
				newSecret("target"),
			},
			reason: "No new Secrets created",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.objects = append(test.objects, &test.ingress)

			// Create a client and the reconciler
			fakeClient := fake.NewClientBuilder().WithObjects(test.objects...).Build()
			reconciler := newReconciler(fakeClient, "source")

			// Reconcile and check for errors
			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: test.ingress.Namespace,
					Name:      test.ingress.Name,
				},
			}

			_, err := reconciler.Reconcile(context.TODO(), request)
			assert.NoError(t, err)

			// Check if the target Secret was created
			if !reflect.ValueOf(test.newSecret).IsZero() {
				newSecretName := types.NamespacedName{
					Namespace: test.newSecret.ObjectMeta.Namespace,
					Name:      test.newSecret.ObjectMeta.Name,
				}

				var newSecret corev1.Secret
				err := fakeClient.Get(context.TODO(), newSecretName, &newSecret)

				assert.NoError(t, err)
				assert.Equal(t, corev1.SecretTypeTLS, newSecret.Type)
				assert.Equal(t, "certificate", string(newSecret.Data[corev1.TLSCertKey]))
				assert.Equal(t, "private key", string(newSecret.Data[corev1.TLSPrivateKeyKey]))
			}
		})
	}
}
