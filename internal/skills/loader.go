package skills

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-service/internal/skill"
)

// LoadFromDir membaca semua file *.manifest.json dari direktori,
// lalu mencari script pasangannya (.py, .js, .sh) dengan nama yang sama.
func LoadFromDir(dir string, logger *slog.Logger, timeout time.Duration) ([]skill.Skill, error) {
	if dir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read custom skills dir %q: %w", dir, err)
	}

	var loaded []skill.Skill
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".manifest.json") {
			continue
		}

		base := strings.TrimSuffix(entry.Name(), ".manifest.json")
		manifestPath := filepath.Join(dir, entry.Name())

		m, err := loadManifest(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("load manifest %s: %w", entry.Name(), err)
		}

		scriptPath := findScript(dir, base)
		if scriptPath == "" {
			return nil, fmt.Errorf(
				"manifest %s ditemukan tapi tidak ada script-nya (%s.py / .js / .sh)",
				entry.Name(), base,
			)
		}

		loaded = append(loaded, NewScriptSkill(*m, scriptPath, timeout))
		logger.Info("custom skill loaded", "name", m.Name, "script", filepath.Base(scriptPath))
	}
	return loaded, nil
}

func loadManifest(path string) (*skill.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m skill.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Name == "" {
		return nil, fmt.Errorf("manifest %s: field 'name' wajib diisi", path)
	}
	return &m, nil
}

func findScript(dir, base string) string {
	for _, ext := range []string{".py", ".js", ".sh"} {
		p := filepath.Join(dir, base+ext)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
