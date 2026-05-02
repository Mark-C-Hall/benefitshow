package ballot

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

var ErrAlreadyVoted = errors.New("user has already voted")

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("ballot: open %s: %w", path, err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, fmt.Errorf("ballot: journal_mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		return nil, fmt.Errorf("ballot: foreign_keys: %w", err)
	}
	if _, err := db.Exec(`PRAGMA synchronous=NORMAL`); err != nil {
		return nil, fmt.Errorf("ballot: synchronous: %w", err)
	}
	if err := initSchema(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
  id            INTEGER PRIMARY KEY,
  discord_id    TEXT UNIQUE NOT NULL,
  username      TEXT NOT NULL,
  session_token TEXT UNIQUE,
  has_voted     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS ballots (
  id      INTEGER PRIMARY KEY,
  user_id INTEGER REFERENCES users(id),
  source  TEXT NOT NULL CHECK (source IN ('online','paper')),
  rank_1  INTEGER NOT NULL,
  rank_2  INTEGER NOT NULL,
  rank_3  INTEGER NOT NULL,
  rank_4  INTEGER NOT NULL,
  rank_5  INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ballots_user_online
  ON ballots(user_id) WHERE source = 'online' AND user_id IS NOT NULL;
`)
	if err != nil {
		return fmt.Errorf("ballot: init schema: %w", err)
	}
	return nil
}

// EnsureUser upserts a user row and returns the row id. Used by the dev-stub
// bootstrap and (later) the real Discord OAuth callback.
func (s *Store) EnsureUser(discordID, username string) (int64, error) {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO users (discord_id, username) VALUES (?, ?)`,
		discordID, username,
	)
	if err != nil {
		return 0, fmt.Errorf("ballot: ensure user insert: %w", err)
	}
	var id int64
	if err := s.db.QueryRow(`SELECT id FROM users WHERE discord_id = ?`, discordID).Scan(&id); err != nil {
		return 0, fmt.Errorf("ballot: ensure user select: %w", err)
	}
	return id, nil
}

func (s *Store) HasVoted(userID int64) (bool, error) {
	var v int
	err := s.db.QueryRow(`SELECT has_voted FROM users WHERE id = ?`, userID).Scan(&v)
	if err != nil {
		return false, fmt.Errorf("ballot: has voted: %w", err)
	}
	return v == 1, nil
}

// GetOnlineBallot returns the five ranked song IDs for the user's online ballot.
func (s *Store) GetOnlineBallot(userID int64) ([5]int, error) {
	var ranks [5]int
	err := s.db.QueryRow(
		`SELECT rank_1, rank_2, rank_3, rank_4, rank_5
		   FROM ballots WHERE user_id = ? AND source = 'online'`,
		userID,
	).Scan(&ranks[0], &ranks[1], &ranks[2], &ranks[3], &ranks[4])
	if err != nil {
		return ranks, fmt.Errorf("ballot: get online ballot: %w", err)
	}
	return ranks, nil
}

// CreateOnlineBallot inserts an online ballot inside a transaction. Returns
// ErrAlreadyVoted if the user has already submitted.
func (s *Store) CreateOnlineBallot(userID int64, ranks [5]int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ballot: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var hasVoted int
	if err := tx.QueryRow(`SELECT has_voted FROM users WHERE id = ?`, userID).Scan(&hasVoted); err != nil {
		return fmt.Errorf("ballot: check voted: %w", err)
	}
	if hasVoted == 1 {
		return ErrAlreadyVoted
	}

	_, err = tx.Exec(
		`INSERT INTO ballots (user_id, source, rank_1, rank_2, rank_3, rank_4, rank_5)
		 VALUES (?, 'online', ?, ?, ?, ?, ?)`,
		userID, ranks[0], ranks[1], ranks[2], ranks[3], ranks[4],
	)
	if err != nil {
		return fmt.Errorf("ballot: insert ballot: %w", err)
	}

	if _, err := tx.Exec(`UPDATE users SET has_voted = 1 WHERE id = ?`, userID); err != nil {
		return fmt.Errorf("ballot: update has_voted: %w", err)
	}

	return tx.Commit()
}
