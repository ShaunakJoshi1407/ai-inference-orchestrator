package cmd

import (
	"context"
	"fmt"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"

	"github.com/ShaunakJoshi1407/ai-inference-orchestrator/cli/internal/k8s"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete an AI model deployment",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		name := args[0]

		k8sClient, err := k8s.GetClient()
		if err != nil {
			return err
		}

		aiDeploy := &infrav1.AIDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
		}

		err = k8sClient.Delete(context.Background(), aiDeploy)
		if err != nil {
			return err
		}

		fmt.Println("AI deployment deleted:", name)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
