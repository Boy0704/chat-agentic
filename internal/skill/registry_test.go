package skill_test

import (
	"context"
	"testing"

	"agent-service/internal/skill"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stub skill untuk keperluan test
type stubSkill struct {
	name   string
	result skill.Result
	err    error
}

func (s *stubSkill) Manifest() skill.Manifest {
	return skill.Manifest{
		Name:        s.name,
		Description: "stub skill untuk test",
		Parameters: skill.ParameterSchema{
			Type:       "object",
			Properties: map[string]skill.Property{},
		},
	}
}

func (s *stubSkill) Execute(_ context.Context, _ skill.Request) (skill.Result, error) {
	return s.result, s.err
}

func newRegistry() *skill.Registry {
	return skill.NewRegistry(&skill.Dependencies{})
}

func TestRegistry_Register(t *testing.T) {
	reg := newRegistry()

	err := reg.Register(&stubSkill{name: "skill_a"})
	require.NoError(t, err)

	manifests := reg.List()
	assert.Len(t, manifests, 1)
	assert.Equal(t, "skill_a", manifests[0].Name)
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	reg := newRegistry()

	require.NoError(t, reg.Register(&stubSkill{name: "skill_a"}))
	err := reg.Register(&stubSkill{name: "skill_a"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_ToOpenAITools(t *testing.T) {
	reg := newRegistry()
	reg.Register(&stubSkill{name: "skill_a"})
	reg.Register(&stubSkill{name: "skill_b"})

	tools := reg.ToOpenAITools()

	assert.Len(t, tools, 2)
	names := []string{tools[0].Function.Name, tools[1].Function.Name}
	assert.ElementsMatch(t, []string{"skill_a", "skill_b"}, names)
	for _, tool := range tools {
		assert.Equal(t, "function", string(tool.Type))
		assert.NotEmpty(t, tool.Function.Description)
	}
}

func TestRegistry_Execute_Found(t *testing.T) {
	reg := newRegistry()
	reg.Register(&stubSkill{
		name:   "skill_a",
		result: skill.Result{Summary: "ok", Data: "data"},
	})

	result, err := reg.Execute(context.Background(), "skill_a", map[string]any{}, map[string]any{})

	require.NoError(t, err)
	assert.Equal(t, "ok", result.Summary)
}

func TestRegistry_Execute_NotFound(t *testing.T) {
	reg := newRegistry()

	_, err := reg.Execute(context.Background(), "tidak_ada", map[string]any{}, map[string]any{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistry_List_Empty(t *testing.T) {
	reg := newRegistry()
	assert.Empty(t, reg.List())
}
