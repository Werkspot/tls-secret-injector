package ingress

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestHandle(t *testing.T) {
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
			// Create a client and the mutator
			fakeClient := fake.NewClientBuilder().WithObjects(test.objects...).Build()
			mutator := newMutator(fakeClient, "source")

			decoder, _ := admission.NewDecoder(scheme.Scheme)
			_ = mutator.InjectDecoder(decoder)

			// Submit the request and verify the response
			ingressJson, _ := json.Marshal(test.ingress)

			request := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Kind:      metav1.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
					Namespace: test.ingress.ObjectMeta.Namespace,
					Name:      test.ingress.ObjectMeta.Name,
					Object:    runtime.RawExtension{Raw: ingressJson},
				},
			}
			response := mutator.Handle(context.TODO(), request)

			assert.True(t, response.Allowed)
			assert.Equal(t, metav1.StatusReason(test.reason), response.Result.Reason)

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

func newIngress(namespace string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "example-io",
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"example.io"},
					SecretName: "tls-example-io",
				},
			},
		},
	}
}

func newSecret(namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "tls-example-io",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("certificate"),
			corev1.TLSPrivateKeyKey: []byte("private key"),
		},
	}
}
