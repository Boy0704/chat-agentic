package skill

import "context"

type Skill interface {
	Manifest() Manifest
	Execute(ctx context.Context, req Request) (Result, error)
}

type Manifest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  ParameterSchema `json:"parameters"`
}

type ParameterSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

type Request struct {
	Params     map[string]any
	AppContext  map[string]any
	Deps       *Dependencies
}

type Result struct {
	Data    any    `json:"data"`
	Summary string `json:"summary"`
}
