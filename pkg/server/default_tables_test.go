package server

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
)

func TestLoadDefaultTableProfiles(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/default_tables.json"
	require.NoError(t, ensureDefaultTablesConfigFile(path))
	require.NoError(t, os.WriteFile(path, []byte(`{
  "tables": [
    {
      "name": "Medium Table",
      "buy_in_dcr": "0.1",
      "min_players": 2,
      "max_players": 6,
      "small_blind": 10,
      "big_blind": 20,
      "starting_chips": 1000
    }
  ]
}`), 0o600))

	profiles, err := loadDefaultTableProfiles(path)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	require.Equal(t, int64(10_000_000), profiles[0].BuyIn)
	require.Equal(t, "Medium Table", profiles[0].Name)
	require.Equal(t, 1, profiles[0].Count)
	require.Equal(t, defaultTableTimeBank, profiles[0].TimeBank)
	require.Equal(t, defaultTableAutoStart, profiles[0].AutoStartDelay)
	require.Equal(t, defaultTableAutoAdvance, profiles[0].AutoAdvanceDelay)
}

func TestLoadServerConfigUsesDefaultTablesPathForExistingConfig(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "pokerd.conf")
	require.NoError(t, os.WriteFile(confPath, []byte(`datadir=`+dir+`
grpchost=localhost
grpcport=50050
grpccertpath=`+filepath.Join(dir, "server.cert")+`
grpckeypath=`+filepath.Join(dir, "server.key")+`
adaptorsecret=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
`), 0o600))

	cfg, err := LoadServerConfig(dir, "pokerd.conf")
	require.NoError(t, err)
	defer cfg.LogBackend.Close()

	require.Equal(t, filepath.Join(dir, defaultTablesFilename), cfg.DefaultTablesPath)
	_, err = os.Stat(cfg.DefaultTablesPath)
	require.NoError(t, err)
	require.Len(t, cfg.DefaultTables, 9)
}

func TestManagedTablePersistsWithoutCreator(t *testing.T) {
	db, err := NewDatabase(filepath.Join(t.TempDir(), "poker.sqlite"))
	require.NoError(t, err)
	defer db.Close()

	cfg := poker.TableConfig{
		ID:               "managed-table",
		Name:             "Managed Table",
		Source:           managedTableSourceDefault,
		BuyIn:            10_000_000,
		MinPlayers:       2,
		MaxPlayers:       6,
		SmallBlind:       10,
		BigBlind:         20,
		StartingChips:    1000,
		TimeBank:         defaultTableTimeBank,
		AutoStartDelay:   defaultTableAutoStart,
		AutoAdvanceDelay: defaultTableAutoAdvance,
	}
	require.NoError(t, db.UpsertTable(context.Background(), &cfg))

	stored, err := db.GetTable(context.Background(), cfg.ID)
	require.NoError(t, err)
	require.Equal(t, managedTableSourceDefault, stored.Source)
}

func TestDefaultTableManagerCreatesConfiguredTables(t *testing.T) {
	db := NewInMemoryDB()
	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	defer srv.Stop()

	srv.defaultTables = []DefaultTableProfile{testDefaultTableProfile(2)}

	require.NoError(t, srv.initializeDefaultTables())
	require.Eventually(t, func() bool {
		return len(managedTablesForServer(srv)) == 2
	}, time.Second, 20*time.Millisecond)

	for _, table := range managedTablesForServer(srv) {
		cfg := table.GetConfig()
		require.Equal(t, managedTableSourceDefault, cfg.Source)
		require.Empty(t, table.GetUsers())
		require.Equal(t, "Micro Table", cfg.Name)
	}
}

func TestDefaultTableManagerDoesNotDuplicateOnRestart(t *testing.T) {
	db := NewInMemoryDB()
	logBackend := createTestLogBackend()
	defer logBackend.Close()

	profiles := []DefaultTableProfile{testDefaultTableProfile(2)}

	srv1, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	srv1.defaultTables = profiles
	require.NoError(t, srv1.initializeDefaultTables())
	require.Eventually(t, func() bool {
		return len(managedTablesForServer(srv1)) == 2
	}, time.Second, 20*time.Millisecond)
	initialIDs := managedTableIDs(srv1)
	srv1.Stop()

	srv2, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	defer srv2.Stop()
	srv2.defaultTables = profiles
	require.NoError(t, srv2.initializeDefaultTables())
	require.Eventually(t, func() bool {
		return len(managedTablesForServer(srv2)) == 2
	}, time.Second, 20*time.Millisecond)
	require.Equal(t, initialIDs, managedTableIDs(srv2))
}

func TestDefaultTableManagerReplacesRemovedManagedTable(t *testing.T) {
	db := NewInMemoryDB()
	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	defer srv.Stop()

	srv.defaultTables = []DefaultTableProfile{testDefaultTableProfile(1)}
	require.NoError(t, srv.initializeDefaultTables())
	require.Eventually(t, func() bool {
		return len(managedTablesForServer(srv)) == 1
	}, time.Second, 20*time.Millisecond)

	oldID := managedTablesForServer(srv)[0].GetConfig().ID
	require.True(t, srv.maybeRemoveTable(oldID))
	require.Eventually(t, func() bool {
		tables := managedTablesForServer(srv)
		return len(tables) == 1 && tables[0].GetConfig().ID != oldID
	}, 2*time.Second, 20*time.Millisecond)
}

func TestDefaultTableManagerReplacesManagedTableAfterGameEndCleanup(t *testing.T) {
	db := NewInMemoryDB()
	logBackend := createTestLogBackend()
	defer logBackend.Close()

	srv, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	defer srv.Stop()

	srv.defaultTables = []DefaultTableProfile{testDefaultTableProfile(1)}
	require.NoError(t, srv.initializeDefaultTables())
	require.Eventually(t, func() bool {
		return len(managedTablesForServer(srv)) == 1
	}, time.Second, 20*time.Millisecond)

	oldID := managedTablesForServer(srv)[0].GetConfig().ID
	srv.schedulePostGameTableCleanup(oldID)

	require.Eventually(t, func() bool {
		tables := managedTablesForServer(srv)
		return len(tables) == 1 && tables[0].GetConfig().ID != oldID
	}, 3*time.Second, 20*time.Millisecond)
}

func testDefaultTableProfile(count int) DefaultTableProfile {
	profile := DefaultTableProfile{
		Name:            "Micro Table",
		BuyInDCR:        "0.1",
		MinPlayers:      2,
		MaxPlayers:      6,
		SmallBlind:      10,
		BigBlind:        20,
		StartingChips:   1000,
		TimeBankSeconds: int(defaultTableTimeBank / time.Second),
		AutoStartMS:     int(defaultTableAutoStart / time.Millisecond),
		AutoAdvanceMS:   int(defaultTableAutoAdvance / time.Millisecond),
	}
	if err := profile.normalizeAndValidate(); err != nil {
		panic(err)
	}
	profile.Count = count
	return profile
}

func managedTablesForServer(s *Server) []*poker.Table {
	tables := make([]*poker.Table, 0)
	for _, table := range s.getAllTables() {
		if table.GetConfig().Source == managedTableSourceDefault {
			tables = append(tables, table)
		}
	}
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].GetConfig().ID < tables[j].GetConfig().ID
	})
	return tables
}

func managedTableIDs(s *Server) []string {
	tables := managedTablesForServer(s)
	ids := make([]string, 0, len(tables))
	for _, table := range tables {
		ids = append(ids, table.GetConfig().ID)
	}
	return ids
}
