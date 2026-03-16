package cmd

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"

	"github.com/ShaunakJoshi1407/ai-inference-orchestrator/cli/internal/k8s"
)

var logsCmd = &cobra.Command{
	Use:   "logs [model]",
	Short: "Show logs for a deployed model",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		model := args[0]

		k8sClient, err := k8s.GetClient()
		if err != nil {
			return err
		}

		podList := &corev1.PodList{}

		err = k8sClient.List(
			context.Background(),
			podList,
			client.InNamespace("default"),
			client.MatchingLabels{
				"app": model,
			},
		)

		if err != nil {
			return err
		}

		if len(podList.Items) == 0 {
			return fmt.Errorf("no pods found for model %s", model)
		}

		for _, pod := range podList.Items {
			fmt.Println("Pod:", pod.Name)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
