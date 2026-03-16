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
	Use:   "endpoint [model]",
	Short: "Get service endpoint for a deployed AI model",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		model := args[0]

		k8sClient, err := k8s.GetClient()
		if err != nil {
			return err
		}

		service := &corev1.Service{}

		err = k8sClient.Get(
			context.Background(),
			client.ObjectKey{
				Name:      model + "-service",
				Namespace: "default",
			},
			service,
		)

		if err != nil {
			return err
		}

		port := service.Spec.Ports[0].Port

		fmt.Printf(
			"http://%s.%s.svc.cluster.local:%d\n",
			service.Name,
			service.Namespace,
			port,
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(endpointCmd)
}
