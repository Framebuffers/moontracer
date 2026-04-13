package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

/*
GuildDBManager maintains a thread-safe registry of per-guild SQLite databases.

Each guild gets its own database file (<dataDir>/<guildID>.db), created and
migrated on first access.
*/
type GuildDBManager struct {
	mu      sync.RWMutex
	dbs     map[string]*bun.DB
	dataDir string
}

// NewGuildDBManager creates a manager that stores guild databases in dataDir.
func NewGuildDBManager(dataDir string) *GuildDBManager {
	return &GuildDBManager{
		dbs:     make(map[string]*bun.DB),
		dataDir: dataDir,
	}
}

/*
GetOrCreate returns the *bun.DB for the given guild, creating and migrating the database file if it doesn't exist yet.

Uses double-checked locking so the hot path (DB already cached) takes only a shared read lock.
*/
func (m *GuildDBManager) GetOrCreate(guildID string) (*bun.DB, error) {
	// Hot path:
	// 	read lock only.
	m.mu.RLock()
	if db, ok := m.dbs[guildID]; ok {
		m.mu.RUnlock()
		return db, nil
	}
	m.mu.RUnlock()

	// Cold path:
	// 	write lock, double-check.
	m.mu.Lock()
	defer m.mu.Unlock()

	if db, ok := m.dbs[guildID]; ok {
		return db, nil
	}

	db, err := m.openAndMigrate(guildID)
	if err != nil {
		return nil, err
	}
	m.dbs[guildID] = db
	return db, nil
}

/*
InitForGuilds pre-creates and migrates databases for all known guilds in parallel, bounded by NumCPU.

Errors are logged but non-fatal. A guild will retry on its first interaction.
*/
func (m *GuildDBManager) InitForGuilds(guildIDs []string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())

	for _, id := range guildIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(guildID string) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := m.GetOrCreate(guildID); err != nil {
				log.Printf("guild_db_manager: init warning for guild %s: %v", guildID, err)
			}
		}(id)
	}

	wg.Wait()
}

// Close closes all open guild database connections.
func (m *GuildDBManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for guildID, db := range m.dbs {
		if err := db.Close(); err != nil {
			log.Printf("guild_db_manager: failed to close DB for guild %s: %v", guildID, err)
		}
	}
	m.dbs = make(map[string]*bun.DB)
}

func (m *GuildDBManager) openAndMigrate(guildID string) (*bun.DB, error) {
	path := filepath.Join(m.dataDir, guildID+".db")
	sqldb, err := sql.Open(sqliteshim.ShimName, path+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open guild DB %s: %w", guildID, err)
	}
	sqldb.SetMaxOpenConns(1) // SQLite best practice: serialize writes

	if err := sqldb.PingContext(context.Background()); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("ping guild DB %s: %w", guildID, err)
	}

	bunDB := bun.NewDB(sqldb, sqlitedialect.New())

	if err := Migrate(bunDB); err != nil {
		bunDB.Close()
		return nil, fmt.Errorf("migrate guild DB %s: %w", guildID, err)
	}

	if n, err := ScrubOrphanedCampaignPlayers(bunDB); err != nil {
		log.Printf("guild_db_manager: scrub warning for guild %s: %v", guildID, err)
	} else if n > 0 {
		log.Printf("guild_db_manager: scrubbed %d orphaned campaign_player rows for guild %s", n, guildID)
	}

	log.Printf("guild_db_manager: initialized DB for guild %s at %s", guildID, path)
	return bunDB, nil
}
