package main

import (
	"encoding/json"
	"net/http"
)

type Tool struct {
	Name        string
	Description string
	Handler     ToolHandler
}

type ToolHandler func(map[string]interface{}) (interface{}, error)

type Server struct {
	tools map[string]Tool
}

func NewServer() *Server {

	s := &Server{
		tools: map[string]Tool{},
	}

	s.registerTools()

	return s
}

func (s *Server) Start(addr string) error {

	http.HandleFunc("/tools", s.handleTool)
	http.HandleFunc("/tools/list", s.listTools)
	http.HandleFunc("/agent", s.handleAgent)

	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleTool(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Tool string                 `json:"tool"`
		Args map[string]interface{} `json:"args"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tool, ok := s.tools[req.Tool]
	if !ok {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	result, err := tool.Handler(req.Args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"tool":   req.Tool,
		"result": result,
	}

	json.NewEncoder(w).Encode(resp)

}

func (s *Server) listTools(w http.ResponseWriter, r *http.Request) {

	out := []Tool{}

	for _, t := range s.tools {
		out = append(out, Tool{
			Name:        t.Name,
			Description: t.Description,
		})
	}

	json.NewEncoder(w).Encode(out)
}
