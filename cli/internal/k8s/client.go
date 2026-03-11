package k8s

import (
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClient() (client.Client, error) {

	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, err
	}

	err = infrav1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{
		Scheme: scheme.Scheme,
	})
}
