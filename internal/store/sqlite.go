package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	db.SetConnMaxLifetime(0)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Init(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA synchronous = NORMAL;`,
		`CREATE TABLE IF NOT EXISTS runs (
			id TEXT PRIMARY KEY,
			experiment TEXT NOT NULL,
			target TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at_ns INTEGER NOT NULL,
			ended_at_ns INTEGER,
			duration_ns INTEGER NOT NULL,
			interval_ns INTEGER NOT NULL,
			apply INTEGER NOT NULL,
			unsafe_max_duration INTEGER NOT NULL,
			result TEXT NOT NULL DEFAULT '',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			stop_requested INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs(started_at_ns DESC);`,
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			ts_ns INTEGER NOT NULL,
			level TEXT NOT NULL,
			action TEXT NOT NULL,
			result TEXT NOT NULL,
			target TEXT NOT NULL,
			message TEXT NOT NULL DEFAULT '',
			details_json TEXT NOT NULL DEFAULT '{}',
			FOREIGN KEY(run_id) REFERENCES runs(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_events_run_id ON events(run_id);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("initialize sqlite schema: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) CreateRun(ctx context.Context, run RunRecord) error {
	metadataJSON, err := marshalMetadata(run.Metadata)
	if err != nil {
		return err
	}

	var endedAt interface{}
	if run.EndedAt != nil {
		endedAt = run.EndedAt.UnixNano()
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO runs (
			id, experiment, target, status, started_at_ns, ended_at_ns, duration_ns, interval_ns,
			apply, unsafe_max_duration, result, metadata_json, stop_requested
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		run.ID,
		run.Experiment,
		run.Target,
		string(run.Status),
		run.StartedAt.UnixNano(),
		endedAt,
		run.Duration.Nanoseconds(),
		run.Interval.Nanoseconds(),
		boolToInt(run.Apply),
		boolToInt(run.UnsafeMaxDuration),
		run.Result,
		metadataJSON,
		boolToInt(run.StopRequested),
	)
	if err != nil {
		return fmt.Errorf("insert run %q: %w", run.ID, err)
	}

	return nil
}

func (s *SQLiteStore) UpdateRunStatus(ctx context.Context, runID string, status RunStatus, endedAt time.Time, result string) error {
	updated, err := s.db.ExecContext(
		ctx,
		`UPDATE runs SET status = ?, ended_at_ns = ?, result = ? WHERE id = ?;`,
		string(status),
		endedAt.UnixNano(),
		result,
		runID,
	)
	if err != nil {
		return fmt.Errorf("update run status for %q: %w", runID, err)
	}

	rows, err := updated.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result for %q: %w", runID, err)
	}
	if rows == 0 {
		return ErrRunNotFound
	}

	return nil
}

func (s *SQLiteStore) UpdateRunMetadata(ctx context.Context, runID string, metadata map[string]string) error {
	metadataJSON, err := marshalMetadata(metadata)
	if err != nil {
		return err
	}

	updated, err := s.db.ExecContext(
		ctx,
		`UPDATE runs SET metadata_json = ? WHERE id = ?;`,
		metadataJSON,
		runID,
	)
	if err != nil {
		return fmt.Errorf("update run metadata for %q: %w", runID, err)
	}

	rows, err := updated.RowsAffected()
	if err != nil {
		return fmt.Errorf("check metadata update result for %q: %w", runID, err)
	}
	if rows == 0 {
		return ErrRunNotFound
	}

	return nil
}

func (s *SQLiteStore) MarkStopRequested(ctx context.Context, runID string, requested bool) error {
	updated, err := s.db.ExecContext(
		ctx,
		`UPDATE runs SET stop_requested = ? WHERE id = ?;`,
		boolToInt(requested),
		runID,
	)
	if err != nil {
		return fmt.Errorf("mark stop requested for %q: %w", runID, err)
	}

	rows, err := updated.RowsAffected()
	if err != nil {
		return fmt.Errorf("check stop update result for %q: %w", runID, err)
	}
	if rows == 0 {
		return ErrRunNotFound
	}

	return nil
}

func (s *SQLiteStore) IsStopRequested(ctx context.Context, runID string) (bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT stop_requested FROM runs WHERE id = ?;`, runID)

	var stopRequested int
	if err := row.Scan(&stopRequested); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrRunNotFound
		}
		return false, fmt.Errorf("read stop request for %q: %w", runID, err)
	}

	return stopRequested == 1, nil
}

func (s *SQLiteStore) GetRun(ctx context.Context, runID string) (RunRecord, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT
			id, experiment, target, status, started_at_ns, ended_at_ns,
			duration_ns, interval_ns, apply, unsafe_max_duration, result, metadata_json, stop_requested
		FROM runs
		WHERE id = ?;`,
		runID,
	)

	run, err := scanRunRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RunRecord{}, ErrRunNotFound
		}
		return RunRecord{}, fmt.Errorf("get run %q: %w", runID, err)
	}

	return run, nil
}

func (s *SQLiteStore) ListRuns(ctx context.Context, limit int) ([]RunRecord, error) {
	if limit <= 0 {
		limit = DefaultStatusLimit
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			id, experiment, target, status, started_at_ns, ended_at_ns,
			duration_ns, interval_ns, apply, unsafe_max_duration, result, metadata_json, stop_requested
		FROM runs
		ORDER BY started_at_ns DESC
		LIMIT ?;`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	runs := make([]RunRecord, 0, limit)
	for rows.Next() {
		run, scanErr := scanRunRows(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan listed runs: %w", scanErr)
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate listed runs: %w", err)
	}

	return runs, nil
}

func (s *SQLiteStore) AddEvent(ctx context.Context, event EventRecord) error {
	detailsJSON, err := marshalMetadata(event.Details)
	if err != nil {
		return err
	}

	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}

	if event.Level == "" {
		event.Level = "info"
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO events (run_id, ts_ns, level, action, result, target, message, details_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
		event.RunID,
		event.Time.UnixNano(),
		event.Level,
		event.Action,
		event.Result,
		event.Target,
		event.Message,
		detailsJSON,
	)
	if err != nil {
		return fmt.Errorf("insert event for run %q: %w", event.RunID, err)
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRunRow(scanner rowScanner) (RunRecord, error) {
	return scanRun(scanner)
}

func scanRunRows(scanner rowScanner) (RunRecord, error) {
	return scanRun(scanner)
}

func scanRun(scanner rowScanner) (RunRecord, error) {
	var (
		run               RunRecord
		status            string
		startedAtNS       int64
		endedAtNSNullable sql.NullInt64
		durationNS        int64
		intervalNS        int64
		applyInt          int
		unsafeInt         int
		metadataJSON      string
		stopRequestedInt  int
	)

	if err := scanner.Scan(
		&run.ID,
		&run.Experiment,
		&run.Target,
		&status,
		&startedAtNS,
		&endedAtNSNullable,
		&durationNS,
		&intervalNS,
		&applyInt,
		&unsafeInt,
		&run.Result,
		&metadataJSON,
		&stopRequestedInt,
	); err != nil {
		return RunRecord{}, err
	}

	run.Status = RunStatus(status)
	run.StartedAt = time.Unix(0, startedAtNS).UTC()
	if endedAtNSNullable.Valid {
		ended := time.Unix(0, endedAtNSNullable.Int64).UTC()
		run.EndedAt = &ended
	}
	run.Duration = time.Duration(durationNS)
	run.Interval = time.Duration(intervalNS)
	run.Apply = applyInt == 1
	run.UnsafeMaxDuration = unsafeInt == 1
	run.StopRequested = stopRequestedInt == 1

	metadata, err := unmarshalMetadata(metadataJSON)
	if err != nil {
		return RunRecord{}, err
	}
	run.Metadata = metadata

	return run, nil
}

func marshalMetadata(metadata map[string]string) (string, error) {
	if len(metadata) == 0 {
		return "{}", nil
	}

	encoded, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("marshal metadata: %w", err)
	}

	return string(encoded), nil
}

func unmarshalMetadata(metadataJSON string) (map[string]string, error) {
	if metadataJSON == "" {
		return map[string]string{}, nil
	}

	metadata := make(map[string]string)
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	return metadata, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
