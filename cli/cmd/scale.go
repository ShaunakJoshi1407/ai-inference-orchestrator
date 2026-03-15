package cmd

import (
	"context"
	"fmt"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"

	"github.com/ShaunakJoshi1407/ai-inference-orchestrator/pkg/k8s"
)

var scaleReplicas int32

var scaleCmd = &cobra.Command{
	Use:   "scale [name]",
	Short: "Scale an AI deployment",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		name := args[0]

		k8sClient, err := k8s.GetClient()
		if err != nil {
			return err
		}

		aiDeploy := &infrav1.AIDeployment{}

		err = k8sClient.Get(
			context.Background(),
			client.ObjectKey{
				Name:      name,
				Namespace: "default",
			},
			aiDeploy,
		)
		if err != nil {
			return err
		}

		aiDeploy.Spec.Replicas = &scaleReplicas

		err = k8sClient.Update(context.Background(), aiDeploy)
		if err != nil {
			return err
		}

		fmt.Printf("Scaled %s to %d replicas\n", name, scaleReplicas)

		return nil
	},
}

func init() {

	scaleCmd.Flags().Int32VarP(
		&scaleReplicas,
		"replicas",
		"r",
		1,
		"Number of replicas",
	)

	rootCmd.AddCommand(scaleCmd)
}
