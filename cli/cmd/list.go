package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"

	"github.com/ShaunakJoshi1407/ai-inference-orchestrator/cli/internal/k8s"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List AI model deployments",

	RunE: func(cmd *cobra.Command, args []string) error {

		k8sClient, err := k8s.GetClient()
		if err != nil {
			return err
		}

		aiDeployList := &infrav1.AIDeploymentList{}

		err = k8sClient.List(context.Background(), aiDeployList, &client.ListOptions{
			Namespace: metav1.NamespaceDefault,
		})
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 10, 4, 2, ' ', 0)

		fmt.Fprintln(w, "NAME\tMODEL\tREPLICAS\tAVAILABLE")

		for _, d := range aiDeployList.Items {

			replicas := int32(0)
			if d.Spec.Replicas != nil {
				replicas = *d.Spec.Replicas
			}

			fmt.Fprintf(
				w,
				"%s\t%s\t%d\t%d\n",
				d.Name,
				d.Spec.Model,
				replicas,
				d.Status.AvailableReplicas,
			)
		}

		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
