package cmd

import (
	"context"
	"fmt"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
)

type ModelConfig struct {
	Image string
	Port  int32
}

var modelRegistry = map[string]ModelConfig{
	"llama3": {
		Image: "ollama/ollama:latest",
		Port:  11434,
	},
	"mistral": {
		Image: "ollama/ollama:latest",
		Port:  11434,
	},
	"phi3": {
		Image: "ollama/ollama:latest",
		Port:  11434,
	},
}

var deployReplicas int32

var deployCmd = &cobra.Command{
	Use:   "deploy [model]",
	Short: "Deploy an AI model using platform defaults",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		model := args[0]

		configEntry, exists := modelRegistry[model]
		if !exists {
			return fmt.Errorf("model %s not supported in registry", model)
		}

		config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return err
		}

		err = infrav1.AddToScheme(scheme.Scheme)
		if err != nil {
			return err
		}

		k8sClient, err := client.New(config, client.Options{
			Scheme: scheme.Scheme,
		})
		if err != nil {
			return err
		}

		replicas := deployReplicas
		port := configEntry.Port

		aiDeploy := &infrav1.AIDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      model,
				Namespace: "default",
			},
			Spec: infrav1.AIDeploymentSpec{
				Model:    model,
				Replicas: &replicas,
				Port:     &port,
			},
		}

		err = k8sClient.Create(context.Background(), aiDeploy)
		if err != nil {
			return err
		}

		fmt.Printf("Model deployed: %s\n", model)
		fmt.Printf("Runtime image: %s\n", configEntry.Image)
		fmt.Printf("Replicas: %d\n", replicas)

		return nil
	},
}

func init() {

	deployCmd.Flags().Int32VarP(
		&deployReplicas,
		"replicas",
		"r",
		1,
		"Number of replicas",
	)

	rootCmd.AddCommand(deployCmd)
}
