package db

import (
	"os"
	"time"

	"github.com/charmbracelet/log"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Db struct {
	Conn *gorm.DB
}

var SQLITE_PRAGMAS = []string{
	`PRAGMA foreign_keys = ON;`,
	`PRAGMA journal_mode = WAL;`,
	`PRAGMA synchronous = NORMAL;`,
	`PRAGMA busy_timeout = 5000;`,
	`PRAGMA temp_store = MEMORY;`,
	`PRAGMA mmap_size = 134217728;`,
	`PRAGMA journal_size_limit = 67108864;`,
	`PRAGMA cache_size = 2000;`,
}

func New(config *config.Config) *Db {
	homeDir, _ := os.UserHomeDir()

	gormConfig := &gorm.Config{
		SkipDefaultTransaction: true,
		TranslateError:         true,
		PrepareStmt:            true,
	}
	if !config.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(sqlite.Open(homeDir+"/.portr/db.sqlite"), gormConfig)
	if err != nil {
		log.Fatal("Failed to connect database", "error", err)
	}

	conn, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database connection", "error", err)
	}

	for _, pragma := range SQLITE_PRAGMAS {
		_, err = conn.Exec(pragma)
		if err != nil {
			log.Fatal("Failed to set pragma", "error", err)
		}
	}

	db.AutoMigrate(&Request{}, &WebSocketSession{}, &WebSocketEvent{})

	return &Db{
		Conn: db,
	}
}

type Request struct {
	ID                 string `gorm:"primaryKey"`
	Subdomain          string
	Localport          int
	Host               string
	Url                string
	Method             string
	Headers            datatypes.JSON
	Body               []byte
	ResponseHeaders    datatypes.JSON
	ResponseBody       []byte
	ResponseStatusCode int
	LoggedAt           time.Time
	IsReplayed         bool
	ParentID           string
	DurationMs         int64
	BytesIn            int64
	BytesOut           int64
	Protocol           string
}

type WebSocketSession struct {
	ID                 string `gorm:"primaryKey"`
	HandshakeRequestID string `gorm:"index"`
	Subdomain          string `gorm:"index:idx_websocket_sessions_tunnel,priority:1"`
	Localport          int    `gorm:"index:idx_websocket_sessions_tunnel,priority:2"`
	Host               string
	URL                string
	Method             string
	RequestHeaders     datatypes.JSON
	ResponseStatusCode int
	ResponseHeaders    datatypes.JSON
	StartedAt          time.Time `gorm:"index"`
	LastEventAt        *time.Time
	ClosedAt           *time.Time
	CloseCode          *int
	CloseReason        string
	EventCount         int64
	ClientEventCount   int64
	ServerEventCount   int64
}

type WebSocketEvent struct {
	ID            string `gorm:"primaryKey"`
	SessionID     string `gorm:"index"`
	Direction     string
	Opcode        int
	OpcodeName    string
	IsFinal       bool
	Payload       []byte
	PayloadLength int
	LoggedAt      time.Time `gorm:"index"`
}

func (d *Db) DeleteRequestsOlderThan(cutoff time.Time) (int64, error) {
	tx := d.Conn.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	var deletedTotal int64
	var sessionIDs []string
	if err := tx.Model(&WebSocketSession{}).
		Where("closed_at IS NOT NULL AND COALESCE(last_event_at, started_at) < ?", cutoff).
		Pluck("id", &sessionIDs).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if len(sessionIDs) > 0 {
		result := tx.Where("session_id IN ?", sessionIDs).Delete(&WebSocketEvent{})
		if result.Error != nil {
			tx.Rollback()
			return 0, result.Error
		}
		deletedTotal += result.RowsAffected
	}

	sessionResult := tx.
		Where("closed_at IS NOT NULL AND COALESCE(last_event_at, started_at) < ?", cutoff).
		Delete(&WebSocketSession{})
	if sessionResult.Error != nil {
		tx.Rollback()
		return 0, sessionResult.Error
	}
	deletedTotal += sessionResult.RowsAffected

	requestResult := tx.Where("logged_at < ?", cutoff).Delete(&Request{})
	if requestResult.Error != nil {
		tx.Rollback()
		return 0, requestResult.Error
	}
	deletedTotal += requestResult.RowsAffected

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return deletedTotal, nil
}
