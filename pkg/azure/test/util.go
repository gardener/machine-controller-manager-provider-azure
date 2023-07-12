package test

import (
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateProviderSecret() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			api.ClientID:       []byte(ClientID),
			api.ClientSecret:   []byte(ClientSecret),
			api.SubscriptionID: []byte(SubscriptionID),
			api.TenantID:       []byte(TenantID),
		},
	}
}
