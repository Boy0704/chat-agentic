package skill

import (
	"context"
	"fmt"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

type Registry struct {
	skills map[string]Skill
	deps   *Dependencies
	mu     sync.RWMutex
}

func NewRegistry(deps *Dependencies) *Registry {
	return &Registry{
		skills: make(map[string]Skill),
		deps:   deps,
	}
}

func (r *Registry) Register(s Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := s.Manifest().Name
	if _, exists := r.skills[name]; exists {
		return fmt.Errorf("skill %q already registered", name)
	}
	r.skills[name] = s
	return nil
}

func (r *Registry) ToOpenAITools() []openai.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]openai.Tool, 0, len(r.skills))
	for _, s := range r.skills {
		m := s.Manifest()
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        m.Name,
				Description: m.Description,
				Parameters:  m.Parameters,
			},
		})
	}
	return tools
}

func (r *Registry) Execute(ctx context.Context, name string, params map[string]any, appCtx map[string]any) (Result, error) {
	r.mu.RLock()
	s, ok := r.skills[name]
	r.mu.RUnlock()

	if !ok {
		return Result{}, fmt.Errorf("skill %q not found", name)
	}

	return s.Execute(ctx, Request{
		Params:    params,
		AppContext: appCtx,
		Deps:      r.deps,
	})
}

func (r *Registry) List() []Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	manifests := make([]Manifest, 0, len(r.skills))
	for _, s := range r.skills {
		manifests = append(manifests, s.Manifest())
	}
	return manifests
}
