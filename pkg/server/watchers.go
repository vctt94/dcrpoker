package server

import "sync"

type watcherRegistry struct {
	mu       sync.RWMutex
	byTable  map[string]map[string]struct{}
	byPlayer map[string]map[string]struct{}
}

func (wr *watcherRegistry) add(tableID, playerID string) bool {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	if wr.byTable == nil {
		wr.byTable = make(map[string]map[string]struct{})
	}
	if wr.byPlayer == nil {
		wr.byPlayer = make(map[string]map[string]struct{})
	}

	tableWatchers := wr.byTable[tableID]
	if tableWatchers == nil {
		tableWatchers = make(map[string]struct{})
		wr.byTable[tableID] = tableWatchers
	}
	if _, exists := tableWatchers[playerID]; exists {
		return false
	}
	tableWatchers[playerID] = struct{}{}

	playerTables := wr.byPlayer[playerID]
	if playerTables == nil {
		playerTables = make(map[string]struct{})
		wr.byPlayer[playerID] = playerTables
	}
	playerTables[tableID] = struct{}{}
	return true
}

func (wr *watcherRegistry) remove(tableID, playerID string) bool {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	tableWatchers := wr.byTable[tableID]
	if tableWatchers == nil {
		return false
	}
	if _, exists := tableWatchers[playerID]; !exists {
		return false
	}
	delete(tableWatchers, playerID)
	if len(tableWatchers) == 0 {
		delete(wr.byTable, tableID)
	}

	playerTables := wr.byPlayer[playerID]
	if playerTables != nil {
		delete(playerTables, tableID)
		if len(playerTables) == 0 {
			delete(wr.byPlayer, playerID)
		}
	}
	return true
}

func (wr *watcherRegistry) removeAllForPlayer(playerID string) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	playerTables := wr.byPlayer[playerID]
	if playerTables == nil {
		return
	}
	for tableID := range playerTables {
		tableWatchers := wr.byTable[tableID]
		delete(tableWatchers, playerID)
		if len(tableWatchers) == 0 {
			delete(wr.byTable, tableID)
		}
	}
	delete(wr.byPlayer, playerID)
}

func (wr *watcherRegistry) removeAllForTable(tableID string) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	tableWatchers := wr.byTable[tableID]
	if tableWatchers == nil {
		return
	}
	for playerID := range tableWatchers {
		playerTables := wr.byPlayer[playerID]
		delete(playerTables, tableID)
		if len(playerTables) == 0 {
			delete(wr.byPlayer, playerID)
		}
	}
	delete(wr.byTable, tableID)
}

func (wr *watcherRegistry) watchersForTable(tableID string) []string {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	tableWatchers := wr.byTable[tableID]
	if len(tableWatchers) == 0 {
		return nil
	}
	watchers := make([]string, 0, len(tableWatchers))
	for playerID := range tableWatchers {
		watchers = append(watchers, playerID)
	}
	return watchers
}

func (wr *watcherRegistry) isWatching(tableID, playerID string) bool {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	tableWatchers := wr.byTable[tableID]
	if tableWatchers == nil {
		return false
	}
	_, ok := tableWatchers[playerID]
	return ok
}

func (s *Server) addTableWatcher(tableID, playerID string) bool {
	return s.watchers.add(tableID, playerID)
}

func (s *Server) removeTableWatcher(tableID, playerID string) bool {
	return s.watchers.remove(tableID, playerID)
}

func (s *Server) removeAllWatchersForTable(tableID string) {
	s.watchers.removeAllForTable(tableID)
}

func (s *Server) isTableWatcher(tableID, playerID string) bool {
	return s.watchers.isWatching(tableID, playerID)
}

func (s *Server) tableAudience(tableID string, seated []string) []string {
	watchers := s.watchers.watchersForTable(tableID)
	if len(watchers) == 0 {
		return seated
	}

	seen := make(map[string]struct{}, len(seated)+len(watchers))
	audience := make([]string, 0, len(seated)+len(watchers))
	for _, playerID := range seated {
		if playerID == "" {
			continue
		}
		if _, ok := seen[playerID]; ok {
			continue
		}
		seen[playerID] = struct{}{}
		audience = append(audience, playerID)
	}
	for _, playerID := range watchers {
		if playerID == "" {
			continue
		}
		if _, ok := seen[playerID]; ok {
			continue
		}
		seen[playerID] = struct{}{}
		audience = append(audience, playerID)
	}
	return audience
}
