package main

import (
	"context"
	"fmt"

	infrav1 "github.com/ShaunakJoshi1407/ai-inference-orchestrator/api/v1"
	"github.com/ShaunakJoshi1407/ai-inference-orchestrator/pkg/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Server) registerTools() {

	s.tools["deploy_model"] = Tool{
		Name:        "deploy_model",
		Description: "Deploy a new AI model",
		Handler:     deployModel,
	}

	s.tools["scale_model"] = Tool{
		Name:        "scale_model",
		Description: "Scale an existing model deployment",
		Handler:     scaleModel,
	}

	s.tools["delete_model"] = Tool{
		Name:        "delete_model",
		Description: "Delete a deployed model",
		Handler:     deleteModel,
	}

	s.tools["list_models"] = Tool{
		Name:        "list_models",
		Description: "List deployed models",
		Handler:     listModels,
	}

	s.tools["model_status"] = Tool{
		Name:        "model_status",
		Description: "Get model deployment status",
		Handler:     modelStatus,
	}
}

func modelFromArgs(args map[string]interface{}) (string, error) {
	v, ok := args["model"]
	if !ok {
		return "", fmt.Errorf("model argument is required")
	}
	model, ok := v.(string)
	if !ok || model == "" {
		return "", fmt.Errorf("model must be a non-empty string")
	}
	return model, nil
}

func deployModel(ctx context.Context, args map[string]interface{}) (interface{}, error) {

	model, err := modelFromArgs(args)
	if err != nil {
		return nil, err
	}

	replicas := int32(1)

	if r, ok := args["replicas"]; ok {

		switch v := r.(type) {

		case float64:
			replicas = int32(v)

		case int:
			replicas = int32(v)

		case int32:
			replicas = v

		}

	}

	k8sClient, err := k8s.GetClient()
	if err != nil {
		return nil, err
	}

	port := int32(8080)

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

	if err := k8sClient.Create(ctx, aiDeploy); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"model":    model,
		"replicas": replicas,
		"status":   "deployed",
	}, nil
}

func scaleModel(ctx context.Context, args map[string]interface{}) (interface{}, error) {

	model, err := modelFromArgs(args)
	if err != nil {
		return nil, err
	}

	var replicas int32

	switch v := args["replicas"].(type) {

	case float64:
		replicas = int32(v)

	case int:
		replicas = int32(v)

	case int32:
		replicas = v
	}

	k8sClient, err := k8s.GetClient()
	if err != nil {
		return nil, err
	}

	ai := &infrav1.AIDeployment{}

	if err := k8sClient.Get(ctx, clientKey(model), ai); err != nil {
		return nil, err
	}

	ai.Spec.Replicas = &replicas

	if err := k8sClient.Update(ctx, ai); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"model":    model,
		"replicas": replicas,
		"status":   "scaled",
	}, nil
}

func deleteModel(ctx context.Context, args map[string]interface{}) (interface{}, error) {

	model, err := modelFromArgs(args)
	if err != nil {
		return nil, err
	}

	k8sClient, err := k8s.GetClient()
	if err != nil {
		return nil, err
	}

	ai := &infrav1.AIDeployment{}

	if err := k8sClient.Get(ctx, clientKey(model), ai); err != nil {
		return nil, err
	}

	if err := k8sClient.Delete(ctx, ai); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"model":  model,
		"status": "deleted",
	}, nil
}

func listModels(ctx context.Context, _ map[string]interface{}) (interface{}, error) {

	k8sClient, err := k8s.GetClient()
	if err != nil {
		return nil, err
	}

	list := &infrav1.AIDeploymentList{}

	if err := k8sClient.List(ctx, list); err != nil {
		return nil, err
	}

	models := []string{}

	for _, m := range list.Items {
		models = append(models, m.Name)
	}

	return models, nil
}

func modelStatus(ctx context.Context, args map[string]interface{}) (interface{}, error) {

	model, err := modelFromArgs(args)
	if err != nil {
		return nil, err
	}

	k8sClient, err := k8s.GetClient()
	if err != nil {
		return nil, err
	}

	ai := &infrav1.AIDeployment{}

	if err := k8sClient.Get(ctx, clientKey(model), ai); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"model":     model,
		"replicas":  ai.Status.Replicas,
		"available": ai.Status.AvailableReplicas,
	}, nil
}

func clientKey(name string) client.ObjectKey {
	return client.ObjectKey{
		Name:      name,
		Namespace: "default",
	}
}
