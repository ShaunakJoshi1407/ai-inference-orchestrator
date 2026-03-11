package cmd

import (
	"context"
	"fmt"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"

	"github.com/ShaunakJoshi1407/ai-inference-orchestrator/cli/internal/k8s"
)

var replicas int32

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new AI model deployment",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		name := args[0]

		k8sClient, err := k8s.GetClient()
		if err != nil {
			return err
		}

		port := int32(8080)
		aiDeploy := &infrav1.AIDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: infrav1.AIDeploymentSpec{
				Model:    name,
				Replicas: &replicas,
				Port:     &port,
			},
		}

		err = k8sClient.Create(context.Background(), aiDeploy)
		if err != nil {
			return err
		}

		fmt.Println("AI deployment created:", name)

		return nil
	},
}

func init() {

	createCmd.Flags().Int32VarP(
		&replicas,
		"replicas",
		"r",
		1,
		"Number of replicas",
	)

	rootCmd.AddCommand(createCmd)
}
