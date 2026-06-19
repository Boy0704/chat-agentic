package session

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	openai "github.com/sashabaranov/go-openai"
)

type Store struct {
	db *sqlx.DB
}

type MessageRow struct {
	Role      string    `db:"role"       json:"role"`
	Content   string    `db:"content"    json:"content"`
	CreatedAt time.Time `db:"created_at" json:"timestamp"`
}

func NewStore(db *sqlx.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("session migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id         TEXT     NOT NULL,
			role       TEXT     NOT NULL,
			content    TEXT     NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_id ON sessions(id);
	`)
	return err
}

func (s *Store) Get(ctx context.Context, sessionID string) ([]openai.ChatCompletionMessage, error) {
	var rows []MessageRow
	err := s.db.SelectContext(ctx, &rows,
		`SELECT role, content, created_at FROM sessions WHERE id = ? ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}

	messages := make([]openai.ChatCompletionMessage, 0, len(rows))
	for _, r := range rows {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    r.Role,
			Content: r.Content,
		})
	}
	return messages, nil
}

func (s *Store) Append(ctx context.Context, sessionID, userMsg, assistantMsg string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	nowPlus := time.Now().UTC().Add(time.Millisecond).Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, role, content, created_at) VALUES
		(?, 'user',      ?, ?),
		(?, 'assistant', ?, ?)
	`, sessionID, userMsg, now, sessionID, assistantMsg, nowPlus)
	return err
}

func (s *Store) Delete(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE id = ?`, sessionID,
	)
	return err
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) GetHistory(ctx context.Context, sessionID string) ([]MessageRow, error) {
	var rows []MessageRow
	err := s.db.SelectContext(ctx, &rows,
		`SELECT role, content, created_at FROM sessions WHERE id = ? ORDER BY created_at ASC`,
		sessionID,
	)
	return rows, err
}
