package monitor

import (
	"stormsim/internal/common/logger"
	"sync"
	"time"
)

type VisibilityWindow struct {
	Start      time.Time
	End        time.Time
	GroundGnbs []string
}

type SatelliteInfo struct {
	GnbId             string
	OrbitalPeriod     time.Duration
	VisibilityWindows []VisibilityWindow
	LastUpdated       time.Time
}

type NTNCoordinator struct {
	satellites map[string]*SatelliteInfo
	visibility map[string][]VisibilityWindow
	groundGnbs map[string]bool
	mu         sync.RWMutex
	logger     *logger.Logger
}

func NewNTNCoordinator() *NTNCoordinator {
	return &NTNCoordinator{
		satellites: make(map[string]*SatelliteInfo),
		visibility: make(map[string][]VisibilityWindow),
		groundGnbs: make(map[string]bool),
		logger:     logger.InitLogger("ntn-coordinator", map[string]string{"mod": "ntn-coordinator"}),
	}
}

func (nc *NTNCoordinator) RegisterSatellite(gnbId string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.satellites[gnbId] = &SatelliteInfo{
		GnbId:             gnbId,
		VisibilityWindows: make([]VisibilityWindow, 0),
		LastUpdated:       time.Now(),
	}
	nc.logger.Info("Registered satellite gNB: %s", gnbId)
}

func (nc *NTNCoordinator) UnregisterSatellite(gnbId string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	delete(nc.satellites, gnbId)
	delete(nc.visibility, gnbId)
	nc.logger.Info("Unregistered satellite gNB: %s", gnbId)
}

func (nc *NTNCoordinator) RegisterGroundGnb(gnbId string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.groundGnbs[gnbId] = true
	nc.logger.Info("Registered ground gNB for NTN coordination: %s", gnbId)
}

func (nc *NTNCoordinator) UnregisterGroundGnb(gnbId string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	delete(nc.groundGnbs, gnbId)
	nc.logger.Info("Unregistered ground gNB from NTN coordination: %s", gnbId)
}

func (nc *NTNCoordinator) UpdateSatelliteVisibility(gnbId string, windows []VisibilityWindow) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if sat, ok := nc.satellites[gnbId]; ok {
		sat.VisibilityWindows = windows
		sat.LastUpdated = time.Now()
		nc.logger.Info("Updated visibility windows for satellite %s: %d windows", gnbId, len(windows))
	}
}

func (nc *NTNCoordinator) SetOrbitalPeriod(gnbId string, period time.Duration) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if sat, ok := nc.satellites[gnbId]; ok {
		sat.OrbitalPeriod = period
		nc.logger.Info("Set orbital period for satellite %s: %v", gnbId, period)
	}
}

func (nc *NTNCoordinator) IsHandoverPossible(sourceGnbId, targetGnbId string) bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	sourceIsGround := nc.groundGnbs[sourceGnbId]
	targetIsGround := nc.groundGnbs[targetGnbId]

	if sourceIsGround && targetIsGround {
		return true
	}

	var satelliteId string
	var groundGnbId string

	if !sourceIsGround && targetIsGround {
		satelliteId = sourceGnbId
		groundGnbId = targetGnbId
	} else if sourceIsGround && !targetIsGround {
		satelliteId = targetGnbId
		groundGnbId = sourceGnbId
	} else {
		return false
	}

	sat, ok := nc.satellites[satelliteId]
	if !ok {
		return false
	}

	now := time.Now()
	for _, window := range sat.VisibilityWindows {
		if (now.Equal(window.Start) || now.After(window.Start)) && now.Before(window.End) {
			for _, gnb := range window.GroundGnbs {
				if gnb == groundGnbId {
					return true
				}
			}
		}
	}

	return false
}

func (nc *NTNCoordinator) GetVisibleSatellites(groundGnbId string) []string {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	var visible []string
	now := time.Now()

	for satId, sat := range nc.satellites {
		for _, window := range sat.VisibilityWindows {
			if (now.Equal(window.Start) || now.After(window.Start)) && now.Before(window.End) {
				for _, gnb := range window.GroundGnbs {
					if gnb == groundGnbId {
						visible = append(visible, satId)
						break
					}
				}
				break
			}
		}
	}

	return visible
}

func (nc *NTNCoordinator) GetVisibleGroundGnbs(satelliteGnbId string) []string {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	sat, ok := nc.satellites[satelliteGnbId]
	if !ok {
		return nil
	}

	now := time.Now()
	for _, window := range sat.VisibilityWindows {
		if (now.Equal(window.Start) || now.After(window.Start)) && now.Before(window.End) {
			return window.GroundGnbs
		}
	}

	return nil
}

func (nc *NTNCoordinator) PredictNextHandoverWindow(sourceGnbId, targetGnbId string) time.Time {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	sourceIsGround := nc.groundGnbs[sourceGnbId]
	targetIsGround := nc.groundGnbs[targetGnbId]

	if sourceIsGround && targetIsGround {
		return time.Now()
	}

	var satelliteId string
	var groundGnbId string

	if !sourceIsGround && targetIsGround {
		satelliteId = sourceGnbId
		groundGnbId = targetGnbId
	} else if sourceIsGround && !targetIsGround {
		satelliteId = targetGnbId
		groundGnbId = sourceGnbId
	} else {
		return time.Time{}
	}

	sat, ok := nc.satellites[satelliteId]
	if !ok {
		return time.Time{}
	}

	now := time.Now()
	for _, window := range sat.VisibilityWindows {
		if window.End.After(now) {
			for _, gnb := range window.GroundGnbs {
				if gnb == groundGnbId {
					if window.Start.After(now) {
						return window.Start
					}
					return now
				}
			}
		}
	}

	return time.Time{}
}

func (nc *NTNCoordinator) GetSatelliteInfo(gnbId string) *SatelliteInfo {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	if sat, ok := nc.satellites[gnbId]; ok {
		return sat
	}
	return nil
}

func (nc *NTNCoordinator) GetAllSatellites() []string {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	var satellites []string
	for id := range nc.satellites {
		satellites = append(satellites, id)
	}
	return satellites
}

func (nc *NTNCoordinator) GetAllGroundGnbs() []string {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	var gnbs []string
	for id := range nc.groundGnbs {
		gnbs = append(gnbs, id)
	}
	return gnbs
}

func (nc *NTNCoordinator) GetCurrentVisibilityWindow(satelliteGnbId string) *VisibilityWindow {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	sat, ok := nc.satellites[satelliteGnbId]
	if !ok {
		return nil
	}

	now := time.Now()
	for i := range sat.VisibilityWindows {
		window := &sat.VisibilityWindows[i]
		if (now.Equal(window.Start) || now.After(window.Start)) && now.Before(window.End) {
			return window
		}
	}

	return nil
}

func (nc *NTNCoordinator) GetNextVisibilityWindow(satelliteGnbId string) *VisibilityWindow {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	sat, ok := nc.satellites[satelliteGnbId]
	if !ok {
		return nil
	}

	now := time.Now()
	var nextWindow *VisibilityWindow

	for i := range sat.VisibilityWindows {
		window := &sat.VisibilityWindows[i]
		if window.Start.After(now) {
			if nextWindow == nil || window.Start.Before(nextWindow.Start) {
				nextWindow = window
			}
		}
	}

	return nextWindow
}
