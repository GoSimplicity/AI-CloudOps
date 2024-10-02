package constants

import "errors"

var (
	ErrorK8sClientNotReady     = errors.New("k8s client not ready")
	ErrorMetricsClientNotReady = errors.New("metrics client not ready")
)
