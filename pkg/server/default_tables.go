package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/slog"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/statemachine"
)

const (
	managedTableSourceUser    = "user"
	managedTableSourceDefault = "default"
	defaultTablesFilename     = "default_tables.json"
	// TimeBank: how long a player has to act before being auto-folded/checked.
	defaultTableTimeBank = 30 * time.Second
	// AutoStart: delay after showdown before the next hand begins automatically.
	defaultTableAutoStart = 3 * time.Second
	// AutoAdvance: delay before the next street is dealt when all players are all-in.
	defaultTableAutoAdvance      = 3 * time.Second
	defaultSmallBlind            = int64(10)
	defaultBigBlind              = int64(20)
	defaultBlindIncreaseInterval = 5 * time.Minute
	defaultStartingChips         = int64(2000)
)

type defaultTablesFile struct {
	Tables []DefaultTableProfile `json:"tables"`
}

type DefaultTableProfile struct {
	Name                     string `json:"name"`
	BuyInDCR                 string `json:"buy_in_dcr"`
	MinPlayers               int    `json:"min_players"`
	MaxPlayers               int    `json:"max_players"`
	SmallBlind               int64  `json:"small_blind"`
	BigBlind                 int64  `json:"big_blind"`
	StartingChips            int64  `json:"starting_chips"`
	TimeBankSeconds          int    `json:"time_bank_seconds"`
	AutoStartMS              int    `json:"auto_start_ms"`
	AutoAdvanceMS            int    `json:"auto_advance_ms"`
	BlindIncreaseIntervalSec int    `json:"blind_increase_interval_sec,omitempty"`

	BuyIn                 int64         `json:"-"`
	TimeBank              time.Duration `json:"-"`
	AutoStartDelay        time.Duration `json:"-"`
	AutoAdvanceDelay      time.Duration `json:"-"`
	BlindIncreaseInterval time.Duration `json:"-"`
	Count                 int           `json:"-"`
}

//go:embed default_tables_template.json
var defaultTablesTemplate []byte

func ensureDefaultTablesConfigFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create default tables dir: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat default tables config: %w", err)
	}

	if err := os.WriteFile(path, defaultTablesTemplate, 0o600); err != nil {
		return fmt.Errorf("write default tables config: %w", err)
	}
	return nil
}

func loadDefaultTableProfiles(path string) ([]DefaultTableProfile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read default tables config %s: %w", path, err)
	}

	var fileCfg defaultTablesFile
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return nil, fmt.Errorf("decode default tables config %s: %w", path, err)
	}

	profiles := make([]DefaultTableProfile, 0, len(fileCfg.Tables))
	for i := range fileCfg.Tables {
		profile := fileCfg.Tables[i]
		if err := profile.normalizeAndValidate(); err != nil {
			return nil, fmt.Errorf("default table profile %d: %w", i, err)
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func (p *DefaultTableProfile) normalizeAndValidate() error {
	p.Name = strings.TrimSpace(p.Name)
	p.BuyInDCR = strings.TrimSpace(p.BuyInDCR)
	if p.Count == 0 {
		p.Count = 1
	}
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if p.Count < 0 {
		return fmt.Errorf("count must be >= 0")
	}
	if p.MinPlayers <= 0 {
		return fmt.Errorf("min_players must be > 0")
	}
	if p.MaxPlayers <= 0 {
		return fmt.Errorf("max_players must be > 0")
	}
	if p.MinPlayers > p.MaxPlayers {
		return fmt.Errorf("min_players must be <= max_players")
	}
	if p.SmallBlind == 0 {
		p.SmallBlind = defaultSmallBlind
	}
	if p.BigBlind == 0 {
		p.BigBlind = defaultBigBlind
	}
	if p.SmallBlind <= 0 {
		return fmt.Errorf("small_blind must be > 0")
	}
	if p.BigBlind <= 0 {
		return fmt.Errorf("big_blind must be > 0")
	}
	if p.SmallBlind >= p.BigBlind {
		return fmt.Errorf("small_blind must be < big_blind")
	}
	if p.StartingChips == 0 {
		p.StartingChips = defaultStartingChips
	}
	if p.StartingChips < 0 {
		return fmt.Errorf("starting_chips must be > 0")
	}

	if p.BuyInDCR == "" {
		return fmt.Errorf("buy_in_dcr: amount is required")
	}
	parsedAmount, err := strconv.ParseFloat(p.BuyInDCR, 64)
	if err != nil {
		return fmt.Errorf("buy_in_dcr: invalid amount %q", p.BuyInDCR)
	}
	amount, err := dcrutil.NewAmount(parsedAmount)
	if err != nil {
		return fmt.Errorf("buy_in_dcr: %w", err)
	}
	if amount < 0 {
		return fmt.Errorf("buy_in_dcr: amount must be >= 0")
	}
	p.BuyIn = int64(amount)
	p.TimeBank = time.Duration(p.TimeBankSeconds) * time.Second
	p.AutoStartDelay = time.Duration(p.AutoStartMS) * time.Millisecond
	p.AutoAdvanceDelay = time.Duration(p.AutoAdvanceMS) * time.Millisecond
	p.BlindIncreaseInterval = time.Duration(p.BlindIncreaseIntervalSec) * time.Second
	if p.TimeBank == 0 {
		p.TimeBank = defaultTableTimeBank
	}
	if p.AutoStartDelay == 0 {
		p.AutoStartDelay = defaultTableAutoStart
	}
	if p.AutoAdvanceDelay == 0 {
		p.AutoAdvanceDelay = defaultTableAutoAdvance
	}

	return nil
}

type defaultTableManagerStateFn = statemachine.StateFn[defaultTableManager]

type evDefaultTablesStarted struct{}
type evDefaultTableRemoved struct {
	tableID    string
	profileKey string
}
type evDefaultTablesConfigUpdated struct {
	profiles []DefaultTableProfile
}

type defaultTableManager struct {
	server *Server
	log    slog.Logger

	wantedCountByProfile map[string]int
	profileByKey         map[string]DefaultTableProfile
	liveCountByProfile   map[string]int
	profileKeyByTableID  map[string]string
	sm                   *statemachine.Machine[defaultTableManager]
}

func (s *Server) initializeDefaultTables() error {
	if len(s.defaultTables) == 0 {
		return nil
	}
	mgr := newDefaultTableManager(s, s.defaultTables)
	s.defaultTableMgr = mgr
	mgr.start()
	return nil
}

func newDefaultTableManager(s *Server, profiles []DefaultTableProfile) *defaultTableManager {
	mgr := &defaultTableManager{
		server:               s,
		log:                  s.logBackend.Logger("DEFAULT_TABLES"),
		wantedCountByProfile: make(map[string]int),
		profileByKey:         make(map[string]DefaultTableProfile),
		liveCountByProfile:   make(map[string]int),
		profileKeyByTableID:  make(map[string]string),
	}
	mgr.applyConfigUpdate(profiles)
	return mgr
}

func (m *defaultTableManager) start() {
	m.sm = statemachine.New(m, defaultTablesStateBootstrapping, 32)
	m.sm.Start(context.Background())
	m.sm.Send(evDefaultTablesStarted{})
}

func (m *defaultTableManager) stop() {
	if m == nil || m.sm == nil {
		return
	}
	m.sm.Stop()
}

func (m *defaultTableManager) notifyManagedTableRemoved(tableID, profileKey string) {
	if m == nil || m.sm == nil {
		return
	}
	m.sm.TrySend(evDefaultTableRemoved{tableID: tableID, profileKey: profileKey})
}

func defaultTablesStateBootstrapping(m *defaultTableManager, in <-chan any) defaultTableManagerStateFn {
	for ev := range in {
		switch e := ev.(type) {
		case evDefaultTablesStarted:
			m.syncManagedTablesFromServer()
			m.ensureWantedTables()
			return defaultTablesStateRunning
		case evDefaultTablesConfigUpdated:
			m.applyConfigUpdate(e.profiles)
		}
	}
	return nil
}

func defaultTablesStateRunning(m *defaultTableManager, in <-chan any) defaultTableManagerStateFn {
	for ev := range in {
		switch e := ev.(type) {
		case evDefaultTableRemoved:
			m.handleManagedTableRemoved(e)
		case evDefaultTablesConfigUpdated:
			m.applyConfigUpdate(e.profiles)
			m.ensureWantedTables()
		}
	}
	return nil
}

func (m *defaultTableManager) applyConfigUpdate(profiles []DefaultTableProfile) {
	m.wantedCountByProfile = make(map[string]int)
	m.profileByKey = make(map[string]DefaultTableProfile)

	for _, profile := range profiles {
		profileKey := defaultTableProfileKey(profile)
		m.wantedCountByProfile[profileKey] += profile.Count
		if _, ok := m.profileByKey[profileKey]; !ok {
			m.profileByKey[profileKey] = profile
		}
	}
}

func (m *defaultTableManager) syncManagedTablesFromServer() {
	if m == nil || m.server == nil {
		return
	}

	m.liveCountByProfile = make(map[string]int)
	m.profileKeyByTableID = make(map[string]string)

	for _, table := range m.server.getAllTables() {
		cfg := table.GetConfig()
		if cfg.Source != managedTableSourceDefault {
			continue
		}
		m.trackManagedTable(cfg)
	}
}

func (m *defaultTableManager) handleManagedTableRemoved(ev evDefaultTableRemoved) {
	profileKey := strings.TrimSpace(ev.profileKey)
	if trackedKey, ok := m.profileKeyByTableID[ev.tableID]; ok {
		profileKey = trackedKey
		delete(m.profileKeyByTableID, ev.tableID)
	}
	if profileKey == "" {
		return
	}

	if live := m.liveCountByProfile[profileKey]; live > 0 {
		m.liveCountByProfile[profileKey] = live - 1
		if live == 1 {
			delete(m.liveCountByProfile, profileKey)
		}
	}

	m.ensureWantedTablesForProfile(profileKey)
}

func (m *defaultTableManager) ensureWantedTables() {
	for profileKey := range m.wantedCountByProfile {
		m.ensureWantedTablesForProfile(profileKey)
	}
}

func (m *defaultTableManager) ensureWantedTablesForProfile(profileKey string) {
	if m == nil || m.server == nil {
		return
	}

	wanted := m.wantedCountByProfile[profileKey]
	if wanted <= 0 {
		return
	}

	profile, ok := m.profileByKey[profileKey]
	if !ok {
		return
	}

	for m.liveCountByProfile[profileKey] < wanted {
		cfg := defaultTableConfigFromProfile(profile, m.server.logBackend.Logger("TABLE"), m.server.logBackend.Logger("GAME"))
		if _, err := m.server.createTable(context.Background(), cfg, ""); err != nil {
			m.log.Errorf("failed to create managed table for profile %s: %v", profileKey, err)
			return
		}
		m.trackManagedTable(cfg)
	}
}

func (m *defaultTableManager) trackManagedTable(cfg poker.TableConfig) {
	if cfg.Source != managedTableSourceDefault {
		return
	}

	profileKey := defaultTableConfigProfileKey(cfg)
	m.profileKeyByTableID[cfg.ID] = profileKey
	m.liveCountByProfile[profileKey]++
}

func defaultTableConfigFromProfile(profile DefaultTableProfile, tblLog, gameLog slog.Logger) poker.TableConfig {
	cfg := poker.TableConfig{
		ID:                    newTableID(),
		Name:                  profile.Name,
		Log:                   tblLog,
		GameLog:               gameLog,
		Source:                managedTableSourceDefault,
		BuyIn:                 profile.BuyIn,
		MinPlayers:            profile.MinPlayers,
		MaxPlayers:            profile.MaxPlayers,
		SmallBlind:            profile.SmallBlind,
		BigBlind:              profile.BigBlind,
		StartingChips:         profile.StartingChips,
		TimeBank:              profile.TimeBank,
		AutoStartDelay:        profile.AutoStartDelay,
		AutoAdvanceDelay:      profile.AutoAdvanceDelay,
		BlindIncreaseInterval: profile.BlindIncreaseInterval,
	}
	normalizeTableConfig(&cfg)
	return cfg
}

func defaultTableProfileKey(profile DefaultTableProfile) string {
	return fmt.Sprintf("%s|%d|%d|%d|%d|%d|%d|%d|%d|%d",
		profile.Name,
		profile.BuyIn,
		profile.MinPlayers,
		profile.MaxPlayers,
		profile.SmallBlind,
		profile.BigBlind,
		profile.StartingChips,
		profile.TimeBank.Milliseconds(),
		profile.AutoStartDelay.Milliseconds(),
		profile.AutoAdvanceDelay.Milliseconds(),
	)
}

func defaultTableConfigProfileKey(cfg poker.TableConfig) string {
	return fmt.Sprintf("%s|%d|%d|%d|%d|%d|%d|%d|%d|%d",
		cfg.Name,
		cfg.BuyIn,
		cfg.MinPlayers,
		cfg.MaxPlayers,
		cfg.SmallBlind,
		cfg.BigBlind,
		cfg.StartingChips,
		cfg.TimeBank.Milliseconds(),
		cfg.AutoStartDelay.Milliseconds(),
		cfg.AutoAdvanceDelay.Milliseconds(),
	)
}

func normalizeTableConfig(cfg *poker.TableConfig) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.Source) == "" {
		cfg.Source = managedTableSourceUser
	}
	if strings.TrimSpace(cfg.Name) == "" {
		cfg.Name = fmt.Sprintf("Table %s", cfg.ID[:8])
	}
	if cfg.SmallBlind == 0 {
		cfg.SmallBlind = defaultSmallBlind
	}
	if cfg.BigBlind == 0 {
		cfg.BigBlind = defaultBigBlind
	}
	if cfg.TimeBank == 0 {
		cfg.TimeBank = defaultTableTimeBank
	}
	if cfg.StartingChips == 0 {
		cfg.StartingChips = defaultStartingChips
	}
	if cfg.AutoStartDelay == 0 {
		cfg.AutoStartDelay = defaultTableAutoStart
	}
	if cfg.AutoAdvanceDelay == 0 {
		cfg.AutoAdvanceDelay = defaultTableAutoAdvance
	}
	if cfg.BlindIncreaseInterval == 0 &&
		cfg.SmallBlind == defaultSmallBlind &&
		cfg.BigBlind == defaultBigBlind {
		cfg.BlindIncreaseInterval = defaultBlindIncreaseInterval
	}
}
