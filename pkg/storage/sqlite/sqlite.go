package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"
	"github.com/dromara/carbon/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sgaunet/ekspodlogs/internal/database"
)

//go:embed db/migrations/*.sql
var fs embed.FS

type Storage struct {
	Now     func() time.Time
	db      *sql.DB
	dbFile  string
	queries *database.Queries
	closeOnce sync.Once
	closed    bool
	mu        sync.RWMutex
}

func NewStorage(dbFile string) (*Storage, error) {
	// Configure SQLite connection string for concurrent access
	dbURL := fmt.Sprintf("file:%s?cache=shared&mode=rwc&_journal_mode=WAL&_synchronous=NORMAL&_timeout=5000", dbFile)
	db, err := sql.Open("sqlite3", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	
	// Configure connection pool for concurrent access
	db.SetMaxOpenConns(1)  // SQLite works best with a single connection for writes
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)
	
	return &Storage{
		Now:     time.Now,
		db:      db,
		dbFile:  dbFile,
		queries: database.New(db),
	}, nil
}

func (s *Storage) SetNow(now func() time.Time) {
	s.Now = now
}

func (s *Storage) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed {
			return
		}
		s.closed = true
		if closeErr := s.db.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close SQLite database: %w", closeErr)
		}
	})
	return err
}

func (s *Storage) Init() error {
	u, _ := url.Parse(fmt.Sprintf("sqlite3://%s", s.dbFile))
	db := dbmate.New(u)
	db.FS = fs

	fmt.Println("Migrations:")
	migrations, err := db.FindMigrations()
	if err != nil {
		return fmt.Errorf("failed to find migrations: %w", err)
	}
	for _, m := range migrations {
		fmt.Println(m.Version, m.FilePath)
	}
	db.AutoDumpSchema = false
	err = db.CreateAndMigrate()
	if err != nil {
		return fmt.Errorf("failed to create and migrate database: %w", err)
	}
	return nil
}

func (s *Storage) PurgeAll(ctx context.Context) error {
	if err := s.queries.PurgeAll(ctx); err != nil {
		return fmt.Errorf("failed to purge all logs: %w", err)
	}
	return nil
}

func (s *Storage) PurgeSpecificPeriod(ctx context.Context, profile string, loggroup string, podName string, beginDate *carbon.Carbon, endDate *carbon.Carbon) error {
	podName = "%" + podName + "%"
	err := s.queries.PurgeSpecificPeriod(ctx, database.PurgeSpecificPeriodParams{
		Profile:   profile,
		Loggroup:  loggroup,
		PodName:   podName,
		Begindate: beginDate.StdTime(),
		Enddate:   endDate.StdTime(),
	})
	if err != nil {
		return fmt.Errorf("failed to purge specific period: %w", err)
	}
	return nil
}

func (s *Storage) PurgeSpecificLogPodLogs(ctx context.Context, profile string, loggroup string, podName string) error {
	podName = "%" + podName + "%"
	err := s.queries.PurgeSpecificLogPodLogs(ctx, database.PurgeSpecificLogPodLogsParams{
		Profile:  profile,
		Loggroup: loggroup,
		PodName:  podName,
	})
	if err != nil {
		return fmt.Errorf("failed to purge specific log pod logs: %w", err)
	}
	return nil
}

func (s *Storage) AddLog(ctx context.Context, profile string, loggroup string, eventTime time.Time, podName, containerName, nameSpace, log string) error {
	const maxRetries = 3
	const baseDelay = 10 * time.Millisecond
	
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := s.queries.InsertLog(ctx, database.InsertLogParams{
			EventTime:     eventTime,
			Profile:       profile,
			Loggroup:      loggroup,
			PodName:       podName,
			ContainerName: containerName,
			NamespaceName: nameSpace,
			Log:           log,
		}); err != nil {
			lastErr = err
			// Check if it's a database lock error
			errStr := fmt.Sprintf("%v", err)
			if i < maxRetries-1 && (strings.Contains(errStr, "database is locked") || 
				strings.Contains(errStr, "SQLITE_BUSY") ||
				strings.Contains(errStr, "database lock")) {
				// Wait with exponential backoff before retrying
				delay := baseDelay * time.Duration(1<<uint(i))
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
			return fmt.Errorf("failed to insert log: %w", err)
		}
		return nil
	}
	return fmt.Errorf("failed to insert log after %d retries: %w", maxRetries, lastErr)
}

func (s *Storage) GetLogsOfPod(ctx context.Context, profile string, logGroup string, podName string, beginDate, endDate time.Time) ([]database.Log, error) {
	logs, err := s.queries.GetLogsOfPod(ctx, database.GetLogsOfPodParams{
		Begindate: beginDate,
		Enddate:   endDate,
		Loggroup:  logGroup,
		Profile:   profile,
		PodName:   podName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get logs of pod: %w", err)
	}
	return logs, nil
}

func (s *Storage) GetLogs(ctx context.Context, logGroup string, profile string, podName string, beginDate *carbon.Carbon, endDate *carbon.Carbon) ([]database.Log, error) {
	podName = "%" + podName + "%"
	logs, err := s.queries.GetLogs(ctx, database.GetLogsParams{
		Begindate: beginDate.StdTime(),
		Enddate:   endDate.StdTime(),
		Loggroup:  logGroup,
		Profile:   profile,
		PodName:   podName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	return logs, nil
}
