package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateServiceAllowsKubernetesDefaults(t *testing.T) {
	for _, tt := range []struct {
		name     string
		kind     corev1.ServiceType
		protocol corev1.Protocol
		port     int32
		invalid  bool
	}{
		{"omitted type and protocol", "", "", 80, false},
		{"explicit TCP", corev1.ServiceTypeClusterIP, corev1.ProtocolTCP, 80, false},
		{"explicit UDP", corev1.ServiceTypeNodePort, corev1.ProtocolUDP, 53, false},
		{"unknown type", "bad", corev1.ProtocolTCP, 80, true},
		{"unknown protocol", corev1.ServiceTypeClusterIP, "bad", 80, true},
		{"port out of range", "", "", 65536, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default"}, Spec: corev1.ServiceSpec{Type: tt.kind, Ports: []corev1.ServicePort{{Port: tt.port, Protocol: tt.protocol}}}}
			err := ValidateService(svc)
			if (err != nil) != tt.invalid {
				t.Fatalf("unexpected validation: %v", err)
			}
		})
	}
}
