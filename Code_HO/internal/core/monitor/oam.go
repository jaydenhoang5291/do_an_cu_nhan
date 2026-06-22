package monitor

import (
	"encoding/json"
	"net/http"
	"stormsim/internal/common/logger"
)

type HandoverMonitorOAM struct {
	monitor *HandoverMonitor
	logger  *logger.Logger
}

func NewHandoverMonitorOAM(monitor *HandoverMonitor) *HandoverMonitorOAM {
	return &HandoverMonitorOAM{
		monitor: monitor,
		logger:  logger.InitLogger("handover-oam", map[string]string{"mod": "handover-oam"}),
	}
}

type HandoverStatsResponse struct {
	TotalHandovers   int64   `json:"total_handovers"`
	SuccessHandovers int64   `json:"success_handovers"`
	FailedHandovers  int64   `json:"failed_handovers"`
	SuccessRate      float64 `json:"success_rate"`
	QueueSize        int     `json:"queue_size"`
}

type ScheduledHandoverResponse struct {
	PrUeId       int64  `json:"pr_ue_id"`
	SourceGnbId  string `json:"source_gnb_id"`
	TargetGnbId  string `json:"target_gnb_id"`
	HandoverType int    `json:"handover_type"`
	Priority     int    `json:"priority"`
}

type NTNVisibilityResponse struct {
	SatelliteGnbId    string   `json:"satellite_gnb_id"`
	VisibleGroundGnbs []string `json:"visible_ground_gnbs"`
	NextWindowStart   string   `json:"next_window_start,omitempty"`
	CurrentWindowEnd  string   `json:"current_window_end,omitempty"`
}

type GnbGroupResponse struct {
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	GnbCount         int               `json:"gnb_count"`
	GnbIds           []string          `json:"gnb_ids"`
	NetworkCondition *NetworkCondition `json:"network_condition,omitempty"`
}

func (oam *HandoverMonitorOAM) GetStats(w http.ResponseWriter, r *http.Request) {
	total, success, failed := oam.monitor.GetStats()

	var successRate float64
	if total > 0 {
		successRate = float64(success) / float64(total) * 100.0
	}

	response := HandoverStatsResponse{
		TotalHandovers:   total,
		SuccessHandovers: success,
		FailedHandovers:  failed,
		SuccessRate:      successRate,
		QueueSize:        oam.monitor.GetQueueSize(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

func (oam *HandoverMonitorOAM) GetTrackerResults(w http.ResponseWriter, r *http.Request) {
	tracker := oam.monitor.GetTracker()
	if tracker == nil {
		writeErrorResponse(w, http.StatusNotFound, "tracker not found")
		return
	}

	results := tracker.GetAllResults()
	writeJSONResponse(w, http.StatusOK, results)
}

func (oam *HandoverMonitorOAM) GetTrackerSummary(w http.ResponseWriter, r *http.Request) {
	tracker := oam.monitor.GetTracker()
	if tracker == nil {
		writeErrorResponse(w, http.StatusNotFound, "tracker not found")
		return
	}

	summary := tracker.GetStatsSummary()
	writeJSONResponse(w, http.StatusOK, summary)
}

func (oam *HandoverMonitorOAM) GetQueue(w http.ResponseWriter, r *http.Request) {
	q := oam.monitor.handoverQueue
	if q == nil {
		writeErrorResponse(w, http.StatusNotFound, "queue not found")
		return
	}

	pending := q.GetPendingScheduled()
	response := make([]ScheduledHandoverResponse, len(pending))
	for i, sh := range pending {
		response[i] = ScheduledHandoverResponse{
			PrUeId:       sh.PrUeId,
			SourceGnbId:  sh.SourceGnbId,
			TargetGnbId:  sh.TargetGnbId,
			HandoverType: sh.HandoverType,
			Priority:     sh.Priority,
		}
	}

	writeJSONResponse(w, http.StatusOK, response)
}

func (oam *HandoverMonitorOAM) GetSatellites(w http.ResponseWriter, r *http.Request) {
	ntn := oam.monitor.GetNTNCoordinator()
	if ntn == nil {
		writeErrorResponse(w, http.StatusNotFound, "NTN coordinator not found")
		return
	}

	satellites := ntn.GetAllSatellites()
	writeJSONResponse(w, http.StatusOK, satellites)
}

func (oam *HandoverMonitorOAM) GetGroundGnbs(w http.ResponseWriter, r *http.Request) {
	ntn := oam.monitor.GetNTNCoordinator()
	if ntn == nil {
		writeErrorResponse(w, http.StatusNotFound, "NTN coordinator not found")
		return
	}

	gnbs := ntn.GetAllGroundGnbs()
	writeJSONResponse(w, http.StatusOK, gnbs)
}

func (oam *HandoverMonitorOAM) GetVisibility(w http.ResponseWriter, r *http.Request) {
	ntn := oam.monitor.GetNTNCoordinator()
	if ntn == nil {
		writeErrorResponse(w, http.StatusNotFound, "NTN coordinator not found")
		return
	}

	satelliteGnbId := r.URL.Query().Get("satellite")
	if satelliteGnbId == "" {
		writeErrorResponse(w, http.StatusBadRequest, "satellite parameter required")
		return
	}

	response := NTNVisibilityResponse{
		SatelliteGnbId:    satelliteGnbId,
		VisibleGroundGnbs: ntn.GetVisibleGroundGnbs(satelliteGnbId),
	}

	currentWindow := ntn.GetCurrentVisibilityWindow(satelliteGnbId)
	if currentWindow != nil {
		response.CurrentWindowEnd = currentWindow.End.Format("2006-01-02T15:04:05Z07:00")
	}

	nextWindow := ntn.GetNextVisibilityWindow(satelliteGnbId)
	if nextWindow != nil {
		response.NextWindowStart = nextWindow.Start.Format("2006-01-02T15:04:05Z07:00")
	}

	writeJSONResponse(w, http.StatusOK, response)
}

func (oam *HandoverMonitorOAM) GetGroups(w http.ResponseWriter, r *http.Request) {
	response := oam.monitor.GetAllGroups()
	writeJSONResponse(w, http.StatusOK, response)
}

func (oam *HandoverMonitorOAM) GetGroupByName(w http.ResponseWriter, r *http.Request) {
	groupName := r.URL.Query().Get("name")
	if groupName == "" {
		writeErrorResponse(w, http.StatusBadRequest, "name parameter required")
		return
	}

	group := oam.monitor.GetGroup(groupName)
	if group == nil {
		writeErrorResponse(w, http.StatusNotFound, "group not found")
		return
	}

	response := GnbGroupResponse{
		Name:             group.Name,
		Type:             group.Type.String(),
		GnbCount:         group.Size(),
		GnbIds:           group.GetGnbIds(),
		NetworkCondition: group.GetNetworkCondition(),
	}

	writeJSONResponse(w, http.StatusOK, response)
}

func (oam *HandoverMonitorOAM) GetBestTarget(w http.ResponseWriter, r *http.Request) {
	sourceGnbId := r.URL.Query().Get("source")
	prUeIdStr := r.URL.Query().Get("pr_ue_id")

	if sourceGnbId == "" {
		writeErrorResponse(w, http.StatusBadRequest, "source parameter required")
		return
	}

	var prUeId int64
	if prUeIdStr != "" {
		prUeId = 0
	}

	target := oam.monitor.GetBestHandoverTarget(prUeId, sourceGnbId)
	writeJSONResponse(w, http.StatusOK, map[string]string{"target_gnb_id": target})
}

func (oam *HandoverMonitorOAM) PredictHandoverWindow(w http.ResponseWriter, r *http.Request) {
	sourceGnbId := r.URL.Query().Get("source")
	targetGnbId := r.URL.Query().Get("target")

	if sourceGnbId == "" || targetGnbId == "" {
		writeErrorResponse(w, http.StatusBadRequest, "source and target parameters required")
		return
	}

	nextWindow := oam.monitor.PredictNextHandoverWindow(sourceGnbId, targetGnbId)
	writeJSONResponse(w, http.StatusOK, map[string]string{
		"next_handover_window": nextWindow.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	encoder := jsonEncoder{w}
	encoder.Encode(data)
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(w, statusCode, map[string]string{"error": message})
}

type jsonEncoder struct {
	w http.ResponseWriter
}

func (e jsonEncoder) Encode(v interface{}) error {
	data, err := jsonMarshal(v)
	if err != nil {
		return err
	}
	e.w.Write(data)
	return nil
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
