package cmd

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"

	"github.com/ShaunakJoshi1407/ai-inference-orchestrator/cli/internal/k8s"
)

var endpointCmd = &cobra.Command{
	Use:   "endpoint [name]",
	Short: "Get service endpoint for deployed model",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		name := args[0]

		k8sClient, err := k8s.GetClient()
		if err != nil {
			return err
		}

		service := &corev1.Service{}

		err = k8sClient.Get(
			context.Background(),
			client.ObjectKey{
				Name:      name + "-service",
				Namespace: "default",
			},
			service,
		)
		if err != nil {
			return fmt.Errorf("service not found for model %s", name)
		}

		if len(service.Spec.Ports) == 0 {
			return fmt.Errorf("service has no exposed ports")
		}

		port := service.Spec.Ports[0].Port

		fmt.Println("Model endpoint:")
		fmt.Printf(
			"Cluster DNS: http://%s.%s.svc.cluster.local:%d\n",
			service.Name,
			service.Namespace,
			port,
		)

		fmt.Println()
		fmt.Println("To access locally run:")
		fmt.Printf(
			"kubectl port-forward svc/%s %d:%d\n",
			service.Name,
			port,
			port,
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(endpointCmd)
}
