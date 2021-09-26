package istio_less_ingress_controller

import (
	"context"
	"net/http"

	"k8s.io/client-go/kubernetes"
)

// Cache watches k8s ingresses, and maintains entries based on hostname
// and path.
type Cache interface {
	// MatchRequest matches an HTTP Request to the namespace, service and upstream address.
	MatchRequest(r *http.Request) (namespace, name, addr string, found bool)

	// StartAutoUpdate starts listening to k8s for updates.
	StartAutoUpdate(ctx context.Context, kubeClient *kubernetes.Clientset)
}
