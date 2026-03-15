package main

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	prompt := strings.ToLower(string(body))

	tool, args, err := parsePrompt(prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, ok := s.tools[tool]
	if !ok {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	result, err := t.Handler(args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"prompt": prompt,
		"tool":   tool,
		"result": result,
	}

	json.NewEncoder(w).Encode(resp)
}

func parsePrompt(prompt string) (string, map[string]interface{}, error) {

	args := map[string]interface{}{}

	if strings.Contains(prompt, "deploy") {

		model := extractModel(prompt)

		replicas := extractReplicas(prompt)

		args["model"] = model
		args["replicas"] = replicas

		return "deploy_model", args, nil
	}

	if strings.Contains(prompt, "scale") {

		model := extractModel(prompt)
		replicas := extractReplicas(prompt)

		args["model"] = model
		args["replicas"] = replicas

		return "scale_model", args, nil
	}

	if strings.Contains(prompt, "delete") {

		model := extractModel(prompt)

		args["model"] = model

		return "delete_model", args, nil
	}

	if strings.Contains(prompt, "status") {

		model := extractModel(prompt)

		args["model"] = model

		return "model_status", args, nil
	}

	if strings.Contains(prompt, "list") {

		return "list_models", args, nil
	}

	return "", nil, http.ErrNotSupported
}

func extractModel(prompt string) string {

	words := strings.Fields(prompt)

	ignore := map[string]bool{
		"deploy":   true,
		"scale":    true,
		"delete":   true,
		"status":   true,
		"model":    true,
		"with":     true,
		"to":       true,
		"replicas": true,
		"replica":  true,
		"of":       true,
	}

	for _, w := range words {

		if ignore[w] {
			continue
		}

		// skip numbers
		if _, err := strconv.Atoi(w); err == nil {
			continue
		}

		return w
	}

	return ""
}

func extractReplicas(prompt string) int {

	re := regexp.MustCompile(`\d+`)
	match := re.FindString(prompt)

	if match == "" {
		return 1
	}

	n, _ := strconv.Atoi(match)

	return n
}
