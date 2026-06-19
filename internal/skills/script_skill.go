package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"agent-service/internal/skill"
)

type ScriptSkill struct {
	manifest skill.Manifest
	path     string
}

func NewScriptSkill(manifest skill.Manifest, scriptPath string) *ScriptSkill {
	return &ScriptSkill{manifest: manifest, path: scriptPath}
}

func (s *ScriptSkill) Manifest() skill.Manifest {
	return s.manifest
}

func (s *ScriptSkill) Execute(ctx context.Context, req skill.Request) (skill.Result, error) {
	input, err := json.Marshal(req.Params)
	if err != nil {
		return skill.Result{}, fmt.Errorf("marshal params: %w", err)
	}

	cmd := exec.CommandContext(ctx, interpreter(s.path), s.path)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Env = buildEnv(req.Deps)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return skill.Result{}, fmt.Errorf("script %s: %s", filepath.Base(s.path), string(exitErr.Stderr))
		}
		return skill.Result{}, fmt.Errorf("run script %s: %w", filepath.Base(s.path), err)
	}

	var result skill.Result
	if err := json.Unmarshal(out, &result); err != nil {
		return skill.Result{}, fmt.Errorf("parse output from %s: %w", filepath.Base(s.path), err)
	}
	return result, nil
}

func interpreter(path string) string {
	switch filepath.Ext(path) {
	case ".py":
		return "python3"
	case ".js":
		return "node"
	default:
		return "sh"
	}
}

func buildEnv(deps *skill.Dependencies) []string {
	env := os.Environ()
	if deps.ClientAPI != nil {
		env = append(env,
			"CLIENT_API_BASE_URL="+deps.ClientAPI.BaseURL,
			"CLIENT_API_AUTH="+deps.ClientAPI.AuthHeader,
		)
	}
	return env
}
