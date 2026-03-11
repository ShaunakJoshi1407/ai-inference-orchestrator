package cmd

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/spf13/cobra"

	"github.com/ShaunakJoshi1407/ai-inference-orchestrator/cli/internal/k8s"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var logsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Show logs for a deployed AI model",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		name := args[0]

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
				"app": name,
			},
		)

		if err != nil {
			return err
		}

		if len(podList.Items) == 0 {
			return fmt.Errorf("no pods found for model %s", name)
		}

		pod := podList.Items[0]

		fmt.Println("Pod:", pod.Name)
		fmt.Println()
		fmt.Println("To stream logs run:")
		fmt.Printf("kubectl logs -f %s\n", pod.Name)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
