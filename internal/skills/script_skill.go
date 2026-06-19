package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"agent-service/internal/skill"
)

type ScriptSkill struct {
	manifest skill.Manifest
	path     string
	timeout  time.Duration
}

func NewScriptSkill(manifest skill.Manifest, scriptPath string, timeout time.Duration) *ScriptSkill {
	return &ScriptSkill{manifest: manifest, path: scriptPath, timeout: timeout}
}

func (s *ScriptSkill) Manifest() skill.Manifest {
	return s.manifest
}

func (s *ScriptSkill) Execute(ctx context.Context, req skill.Request) (skill.Result, error) {
	input, err := json.Marshal(req.Params)
	if err != nil {
		return skill.Result{}, fmt.Errorf("marshal params: %w", err)
	}

	execCtx := ctx
	var cancel context.CancelFunc
	if s.timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(execCtx, interpreter(s.path), s.path)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Env = buildEnv(req.Deps)

	out, err := cmd.Output()
	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return skill.Result{}, fmt.Errorf("script %s: timeout after %s", filepath.Base(s.path), s.timeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return skill.Result{}, fmt.Errorf("script %s: %s", filepath.Base(s.path), sanitizeStderr(string(exitErr.Stderr)))
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

// sanitizeStderr strips non-printable characters and truncates stderr to a safe length.
func sanitizeStderr(s string) string {
	const maxLen = 500
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' || (r >= 32 && !unicode.Is(unicode.Cc, r)) {
			b.WriteRune(r)
		}
	}
	result := strings.TrimSpace(b.String())
	if len(result) > maxLen {
		result = result[:maxLen] + "... (truncated)"
	}
	return result
}
