package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"agent-service/internal/agent"
	"agent-service/internal/api"
	"agent-service/internal/config"
	"agent-service/internal/session"
	"agent-service/internal/skill"
	"agent-service/internal/skills"

	"github.com/jmoiron/sqlx"
	openai "github.com/sashabaranov/go-openai"
	_ "modernc.org/sqlite"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid config", "error", err)
		os.Exit(1)
	}

	// SQLite — hanya untuk sessions, bukan data klien
	if err := os.MkdirAll(filepath.Dir(cfg.DB.Path), 0755); err != nil {
		logger.Error("create data dir", "error", err)
		os.Exit(1)
	}
	db, err := sqlx.Open("sqlite", cfg.DB.Path)
	if err != nil {
		logger.Error("open db", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	sessionStore, err := session.NewStore(db)
	if err != nil {
		logger.Error("init session store", "error", err)
		os.Exit(1)
	}

	// ClientAPI — HTTP client untuk memanggil API sistem klien
	clientAPI := &skill.ClientAPI{
		BaseURL:    cfg.ClientAPI.BaseURL,
		AuthHeader: cfg.ClientAPI.AuthHeader,
		HTTP:       &http.Client{Timeout: time.Duration(cfg.ClientAPI.TimeoutSeconds) * time.Second},
	}

	deps := &skill.Dependencies{
		ClientAPI: clientAPI,
		Logger:    logger,
	}

	registry := skill.NewRegistry(deps)

	// Load custom skills dari direktori klien
	customSkills, err := skills.LoadFromDir(cfg.Skills.CustomPath, logger)
	if err != nil {
		logger.Error("load custom skills", "error", err)
		os.Exit(1)
	}
	for _, s := range customSkills {
		if err := registry.Register(s); err != nil {
			logger.Error("register skill", "name", s.Manifest().Name, "error", err)
			os.Exit(1)
		}
	}

	if len(customSkills) == 0 {
		logger.Warn("tidak ada skill yang dimuat — set skills.custom_path di config.yaml")
	}

	llmCfg := openai.DefaultConfig(cfg.LLM.APIKey)
	llmCfg.BaseURL = cfg.LLM.BaseURL
	llmClient := openai.NewClientWithConfig(llmCfg)

	agentSvc := agent.New(llmClient, cfg.LLM.Model, registry, logger)
	handler := api.NewHandler(agentSvc, sessionStore, registry)
	router := api.SetupRouter(handler, cfg.Server.APIKey, cfg.Server.CORS)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		logger.Info("server started", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
