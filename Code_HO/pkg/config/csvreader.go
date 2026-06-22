package config

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// HandoverStep represents a single handover event extracted from CSV
type HandoverStep struct {
	Step         int    // CSV "Bước" column - step index
	FromGnbId    string // Previous step's configured gNB ID
	ToGnbId      string // Target gNB ID (mapped from ue0_BS_ketnoi)
	HandoverType int    // 1=Xn, 2=N2 (from ue0_handover_to_type)
}

// CSVHandoverConfig holds configuration for CSV handover loading
type CSVHandoverConfig struct {
	FilePath     string         // Path to CSV file
	GnbIdMapping map[int]string // CSV gNB ID -> config gNB ID (e.g., 42 -> "000008")
	StepDelayMs  int            // Delay between steps in milliseconds
	FailMode     bool           // Enable failure mode: inject 100% packet loss to trigger timer expiry

	// Simulation params (read from config file)
	PacketLoss   float64
	XnPacketLoss float64
	XnLatencyMs  int
	XnJitterMs   int
}

// csvColumnIndices holds the column indices for required fields
type csvColumnIndices struct {
	step           int
	bsKetnoi       int // ue0_BS_ketnoi
	handover       int // ue0_handover
	handoverToType int // ue0_handover_to_type: 1=Xn, 2=N2
}

// LoadHandoverEventsFromCSV reads a CSV file and extracts handover events
// Returns only steps where ue0_handover=1
func LoadHandoverEventsFromCSV(cfg CSVHandoverConfig) ([]HandoverStep, error) {
	file, err := os.Open(cfg.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file has no data rows")
	}

	// Parse header to find column indices
	indices, err := parseHeader(records[0])
	if err != nil {
		return nil, err
	}

	// Extract handover events
	handoverSteps := []HandoverStep{}
	var previousGnb string

	for i, row := range records[1:] { // Skip header

		// Parse current gNB connection
		bsKetnoi, err := strconv.Atoi(strings.TrimSpace(row[indices.bsKetnoi]))
		if err != nil {
			continue
		}

		// Map CSV gNB ID to config gNB ID
		currentGnb, ok := cfg.GnbIdMapping[bsKetnoi]
		if !ok {
			// Use the numeric ID as string if no mapping provided
			currentGnb = fmt.Sprintf("%06d", bsKetnoi)
		}

		// Check if handover occurs at this step
		handover, err := strconv.Atoi(strings.TrimSpace(row[indices.handover]))
		if err != nil {
			handover = 0
		}

		if handover == 1 {
			// Parse step number
			step, _ := strconv.Atoi(strings.TrimSpace(row[indices.step]))

			// Parse handover type
			hoType, err := strconv.Atoi(strings.TrimSpace(row[indices.handoverToType]))
			if err != nil {
				hoType = 1 // Default to Xn
			}

			// Create handover step
			hs := HandoverStep{
				Step:         step,
				FromGnbId:    previousGnb,
				ToGnbId:      currentGnb,
				HandoverType: hoType,
			}

			// Only add if we have a previous gNB (not first step)
			if previousGnb != "" && previousGnb != currentGnb {
				handoverSteps = append(handoverSteps, hs)
			}
		}

		// Update previous gNB for next iteration
		if i == 0 || handover == 1 {
			previousGnb = currentGnb
		}
	}

	return handoverSteps, nil
}

// parseHeader finds the indices of required columns in the CSV header
func parseHeader(header []string) (csvColumnIndices, error) {
	indices := csvColumnIndices{
		step:           -1,
		bsKetnoi:       -1,
		handover:       -1,
		handoverToType: -1,
	}

	for i, col := range header {
		col = strings.TrimPrefix(col, "\xef\xbb\xbf")
		col = strings.TrimSpace(col)
		switch col {
		case "Bước", "Step":
			indices.step = i
		case "ue0_BS_ketnoi":
			indices.bsKetnoi = i
		case "ue0_handover":
			indices.handover = i
		case "ue0_handover_to_type":
			indices.handoverToType = i
		}
	}

	// Validate required columns exist
	if indices.step == -1 {
		return indices, fmt.Errorf("CSV missing required column: Bước")
	}
	if indices.bsKetnoi == -1 {
		return indices, fmt.Errorf("CSV missing required column: ue0_BS_ketnoi")
	}
	if indices.handover == -1 {
		return indices, fmt.Errorf("CSV missing required column: ue0_handover")
	}
	if indices.handoverToType == -1 {
		return indices, fmt.Errorf("CSV missing required column: ue0_handover_to_type")
	}

	return indices, nil
}

// ParseGnbMapping parses a comma-separated gNB mapping string
// Format: "37:000008,42:000009,59:000010"
func ParseGnbMapping(mappingStr string) (map[int]string, error) {
	mapping := make(map[int]string)

	if mappingStr == "" {
		return mapping, nil
	}

	pairs := strings.Split(mappingStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid mapping format: %s (expected 'csvId:gnbId')", pair)
		}

		csvId, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid CSV gNB ID: %s", parts[0])
		}

		gnbId := strings.TrimSpace(parts[1])
		mapping[csvId] = gnbId
	}

	return mapping, nil
}

// GetHandoverEventType returns model.EventType equivalent based on handover type
// 1 = XnHandover, 2 = N2Handover
func (hs HandoverStep) IsXnHandover() bool {
	return hs.HandoverType == 1
}

func (hs HandoverStep) IsN2Handover() bool {
	return hs.HandoverType == 2
}

type MeasurementCsvConfig struct {
	FilePath    string // Path to measurement CSV file
	StepDelayMs int    // Delay between steps in milliseconds

	// Simulation params (override config file if set)
	PacketLoss      float64
	LatencyMs       int
	JitterMs        int
	XnPacketLoss    float64 // Xn path packet loss 0.0-1.0
	XnLatencyMs     int     // Xn path one-way delay ms
	XnJitterMs      int     // Xn path jitter ms
	T304DurationMs  int     // T304 timer duration (ms) for HO execution, 0 = use default 100ms
	HoCmdMaxRetries int     // Max retries for HO command delivery, 0 = no retry
	FailMode        bool    // Enable failure mode: inject 100% packet loss to trigger timer expiry
	ChoMode         bool    // Enable Conditional Handover (CHO) with UE-side condition evaluation
}

type MeasurementStep struct {
	Timestamp    int                // timestamp column (Bước)
	RsrpValues   map[string]float64 // gnb_id -> RSRP value
	ConnectedGnb string             // connected_gnb column (current gNB)
	FromGnbId    string             // previous step's connected_gnb (HO source)
	ToGnbId      string             // target gNB when HO detected (= ConnectedGnb on change)
	HoTrigger    int                // 1 if connected_gnb changed from previous step
	HandoverType string             // Type column ("Xn" or "N2")
}

type measurementCsvColumn struct {
	timestamp    int
	connectedGnb int
	hoType       int
}

func LoadMeasurementEventsFromCSV(cfg MeasurementCsvConfig) ([]MeasurementStep, error) {
	file, err := os.Open(cfg.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file has no data rows")
	}

	header := records[0]
	colIndices, err := parseMeasurementHeader(header)
	if err != nil {
		return nil, err
	}

	gnbIndices := make(map[int]string)
	for i, col := range header {
		col = strings.TrimSpace(col)
		if strings.HasPrefix(col, "gnb") && strings.HasSuffix(col, "_rsrp") {
			gnbIndices[i] = strings.TrimSuffix(col, "_rsrp")
		}
	}

	var steps []MeasurementStep
	var prevConnectedGnb string
	for _, row := range records[1:] {
		if colIndices.connectedGnb >= len(row) || colIndices.timestamp >= len(row) {
			continue
		}

		timestamp, _ := strconv.Atoi(strings.TrimSpace(row[colIndices.timestamp]))
		connectedGnb := strings.TrimSpace(row[colIndices.connectedGnb])

		hoTrigger := 0
		fromGnbId := ""
		toGnbId := ""
		if prevConnectedGnb != "" && prevConnectedGnb != connectedGnb {
			hoTrigger = 1
			fromGnbId = prevConnectedGnb
			toGnbId = connectedGnb
		}
		prevConnectedGnb = connectedGnb

		rsrpValues := make(map[string]float64)
		for colIdx, gnbId := range gnbIndices {
			if colIdx < len(row) {
				rsrp, _ := strconv.ParseFloat(strings.TrimSpace(row[colIdx]), 64)
				rsrpValues[gnbId] = rsrp
			}
		}

		hoType := "Xn"
		if colIndices.hoType >= 0 && colIndices.hoType < len(row) {
			t := strings.TrimSpace(row[colIndices.hoType])
			if t != "" {
				hoType = t
			}
		}

		steps = append(steps, MeasurementStep{
			Timestamp:    timestamp,
			RsrpValues:   rsrpValues,
			ConnectedGnb: connectedGnb,
			FromGnbId:    fromGnbId,
			ToGnbId:      toGnbId,
			HoTrigger:    hoTrigger,
			HandoverType: hoType,
		})

		fmt.Println(MeasurementStep{
			Timestamp:    timestamp,
			RsrpValues:   rsrpValues,
			ConnectedGnb: connectedGnb,
			FromGnbId:    fromGnbId,
			ToGnbId:      toGnbId,
			HoTrigger:    hoTrigger,
			HandoverType: hoType,
		})
	}

	return steps, nil
}

func parseMeasurementHeader(header []string) (measurementCsvColumn, error) {
	cols := measurementCsvColumn{
		timestamp:    -1,
		connectedGnb: -1,
		hoType:       -1,
	}

	for i, col := range header {
		col = strings.TrimPrefix(col, "\xef\xbb\xbf")
		col = strings.TrimSpace(col)
		switch col {
		case "Bước", "Step":
			cols.timestamp = i
		case "connected_gnb":
			cols.connectedGnb = i
		case "Type":
			cols.hoType = i
		}
	}

	if cols.timestamp == -1 {
		return cols, fmt.Errorf("CSV missing required column: Bước/Step")
	}
	if cols.connectedGnb == -1 {
		return cols, fmt.Errorf("CSV missing required column: connected_gnb")
	}

	return cols, nil
}
