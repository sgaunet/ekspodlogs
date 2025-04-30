package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
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
}

func NewStorage(dbFile string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}
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
	return s.db.Close()
}

func (s *Storage) Init() error {
	u, _ := url.Parse(fmt.Sprintf("sqlite3://%s", s.dbFile))
	db := dbmate.New(u)
	db.FS = fs

	fmt.Println("Migrations:")
	migrations, err := db.FindMigrations()
	if err != nil {
		return err
	}
	for _, m := range migrations {
		fmt.Println(m.Version, m.FilePath)
	}
	db.AutoDumpSchema = false
	err = db.CreateAndMigrate()
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) PurgeAll(ctx context.Context) error {
	return s.queries.PurgeAll(ctx)
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
	return err
}

func (s *Storage) PurgeSpecificLogPodLogs(ctx context.Context, profile string, loggroup string, podName string) error {
	podName = "%" + podName + "%"
	err := s.queries.PurgeSpecificLogPodLogs(ctx, database.PurgeSpecificLogPodLogsParams{
		Profile:  profile,
		Loggroup: loggroup,
		PodName:  podName,
	})
	return err
}

func (s *Storage) AddLog(ctx context.Context, profile string, loggroup string, eventTime time.Time, podName, containerName, nameSpace, log string) error {
	return s.queries.InsertLog(ctx, database.InsertLogParams{
		EventTime:     eventTime,
		Profile:       profile,
		Loggroup:      loggroup,
		PodName:       podName,
		ContainerName: containerName,
		NamespaceName: nameSpace,
		Log:           log,
	})
}

func (s *Storage) GetLogsOfPod(ctx context.Context, profile string, logGroup string, podName string, beginDate, endDate time.Time) ([]database.Log, error) {
	return s.queries.GetLogsOfPod(ctx, database.GetLogsOfPodParams{
		Begindate: beginDate,
		Enddate:   endDate,
		Loggroup:  logGroup,
		Profile:   profile,
		PodName:   podName,
	})
}

func (s *Storage) GetLogs(ctx context.Context, logGroup string, profile string, podName string, beginDate *carbon.Carbon, endDate *carbon.Carbon) ([]database.Log, error) {
	podName = "%" + podName + "%"
	return s.queries.GetLogs(ctx, database.GetLogsParams{
		Begindate: beginDate.StdTime(),
		Enddate:   endDate.StdTime(),
		Loggroup:  logGroup,
		Profile:   profile,
		PodName:   podName,
	})
}
