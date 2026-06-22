package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGnbMapping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[int]string
		wantErr  bool
	}{
		{
			name:  "valid mapping",
			input: "37:000008, 42:000009,59:000010",
			expected: map[int]string{
				37: "000008",
				42: "000009",
				59: "000010",
			},
			wantErr: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: map[int]string{},
			wantErr:  false,
		},
		{
			name:     "invalid format - missing colon",
			input:    "37-000008",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid format - non-numeric CSV ID",
			input:    "abc:000008",
			expected: nil,
			wantErr:  true,
		},
		{
			name:  "single mapping",
			input: "42:000008",
			expected: map[int]string{
				42: "000008",
			},
			wantErr: false,
		},
		{
			name:  "mapping with spaces",
			input: " 37 : 000008 , 42 : 000009 ",
			expected: map[int]string{
				37: "000008",
				42: "000009",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGnbMapping(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGnbMapping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(result) != len(tt.expected) {
					t.Errorf("ParseGnbMapping() len = %d, want %d", len(result), len(tt.expected))
					return
				}
				for k, v := range tt.expected {
					if result[k] != v {
						t.Errorf("ParseGnbMapping() result[%d] = %s, want %s", k, result[k], v)
					}
				}
			}
		})
	}
}

func TestLoadHandoverEventsFromCSV(t *testing.T) {
	// Create a temporary CSV file for testing
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test_handover.csv")

	csvContent := `Bước,ue0_x,ue0_y,ue0_huong,ue0_BS_ketnoi,ue0_prx_hientai,ue0_speed,ue0_speed_eff,ue0_handover,ue0_handover_to_type
0,7012.3,6472.6,1,42,-52.4,44.46,44.46,0,0
1,7012.3,6472.6,1,42,-52.4,44.46,44.46,0,0
2,7012.3,6472.6,1,37,-52.2,44.46,44.46,1,1
3,7012.3,6472.6,1,37,-52.2,44.46,44.46,0,0
4,7012.3,6472.6,1,59,-52.2,44.46,44.46,1,2
`
	err := os.WriteFile(testCSV, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}

	gnbMapping := map[int]string{
		37: "000008",
		42: "000009",
		59: "000010",
	}

	cfg := CSVHandoverConfig{
		FilePath:     testCSV,
		GnbIdMapping: gnbMapping,
		StepDelayMs:  1000,
	}

	steps, err := LoadHandoverEventsFromCSV(cfg)
	if err != nil {
		t.Fatalf("LoadHandoverEventsFromCSV() error = %v", err)
	}

	// Should have 2 handover events (at steps 2 and 4)
	if len(steps) != 2 {
		t.Errorf("LoadHandoverEventsFromCSV() got %d steps, want 2", len(steps))
	}

	// Verify first handover (step 2: from 42 to 37, Xn type)
	if len(steps) > 0 {
		if steps[0].Step != 2 {
			t.Errorf("First handover step = %d, want 2", steps[0].Step)
		}
		if steps[0].ToGnbId != "000008" { // 37 maps to 000008
			t.Errorf("First handover ToGnbId = %s, want 000008", steps[0].ToGnbId)
		}
		if steps[0].HandoverType != 1 { // Xn
			t.Errorf("First handover type = %d, want 1 (Xn)", steps[0].HandoverType)
		}
	}

	// Verify second handover (step 4: from 37 to 59, N2 type)
	if len(steps) > 1 {
		if steps[1].Step != 4 {
			t.Errorf("Second handover step = %d, want 4", steps[1].Step)
		}
		if steps[1].ToGnbId != "000010" { // 59 maps to 000010
			t.Errorf("Second handover ToGnbId = %s, want 000010", steps[1].ToGnbId)
		}
		if steps[1].HandoverType != 2 { // N2
			t.Errorf("Second handover type = %d, want 2 (N2)", steps[1].HandoverType)
		}
	}
}

func TestLoadHandoverEventsFromCSV_MissingColumns(t *testing.T) {
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test_missing_cols.csv")

	// CSV missing ue0_handover column
	csvContent := `Bước,ue0_x,ue0_BS_ketnoi
0,7012.3,42
`
	err := os.WriteFile(testCSV, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}

	cfg := CSVHandoverConfig{
		FilePath:     testCSV,
		GnbIdMapping: map[int]string{},
	}

	_, err = LoadHandoverEventsFromCSV(cfg)
	if err == nil {
		t.Error("LoadHandoverEventsFromCSV() should return error for missing columns")
	}
}

func TestLoadMeasurementEventsFromCSV_BasicChangeDetection(t *testing.T) {
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test_measurement.csv")

	csvContent := `Bước,gnb1_rsrp,connected_gnb,Type
0,-85,000008,Xn
1,-84,000008,Xn
2,-72,000009,Xn
3,-71,000009,Xn
4,-60,000008,Xn
`
	err := os.WriteFile(testCSV, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}

	cfg := MeasurementCsvConfig{
		FilePath:    testCSV,
		StepDelayMs: 100,
	}

	steps, err := LoadMeasurementEventsFromCSV(cfg)
	if err != nil {
		t.Fatalf("LoadMeasurementEventsFromCSV() error = %v", err)
	}

	if len(steps) != 5 {
		t.Fatalf("Expected 5 steps, got %d", len(steps))
	}

	hoSteps := []struct {
		idx       int
		timestamp int
		hoTrigger int
		fromGnbId string
		toGnbId   string
		connected string
	}{
		{0, 0, 0, "", "", "000008"},
		{1, 1, 0, "", "", "000008"},
		{2, 2, 1, "000008", "000009", "000009"},
		{3, 3, 0, "", "", "000009"},
		{4, 4, 1, "000009", "000008", "000008"},
	}

	for _, expected := range hoSteps {
		s := steps[expected.idx]
		if s.Timestamp != expected.timestamp {
			t.Errorf("Step %d: timestamp = %d, want %d", expected.idx, s.Timestamp, expected.timestamp)
		}
		if s.HoTrigger != expected.hoTrigger {
			t.Errorf("Step %d: HoTrigger = %d, want %d", expected.idx, s.HoTrigger, expected.hoTrigger)
		}
		if s.FromGnbId != expected.fromGnbId {
			t.Errorf("Step %d: FromGnbId = %s, want %s", expected.idx, s.FromGnbId, expected.fromGnbId)
		}
		if s.ToGnbId != expected.toGnbId {
			t.Errorf("Step %d: ToGnbId = %s, want %s", expected.idx, s.ToGnbId, expected.toGnbId)
		}
		if s.ConnectedGnb != expected.connected {
			t.Errorf("Step %d: ConnectedGnb = %s, want %s", expected.idx, s.ConnectedGnb, expected.connected)
		}
	}
}

func TestLoadMeasurementEventsFromCSV_IgnoresUeHandoverColumn(t *testing.T) {
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test_ignore_ho.csv")

	csvContent := `Bước,connected_gnb,ue0_handover,ue0_handover_to_type,Type
0,000008,0,000008,Xn
1,000008,1,000009,Xn
2,000008,0,000008,Xn
`
	err := os.WriteFile(testCSV, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}

	cfg := MeasurementCsvConfig{
		FilePath:    testCSV,
		StepDelayMs: 100,
	}

	steps, err := LoadMeasurementEventsFromCSV(cfg)
	if err != nil {
		t.Fatalf("LoadMeasurementEventsFromCSV() error = %v", err)
	}

	if len(steps) != 3 {
		t.Fatalf("Expected 3 steps, got %d", len(steps))
	}

	for i, s := range steps {
		if s.HoTrigger != 0 {
			t.Errorf("Step %d: HoTrigger = %d, want 0 (connected_gnb never changed)", i, s.HoTrigger)
		}
	}
}

func TestLoadMeasurementEventsFromCSV_SamplePattern(t *testing.T) {
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test_sample.csv")

	csvContent := `Bước,gnb1_rsrp,gnb2_rsrp,gnb3_rsrp,gnb4_rsrp,gnb5_rsrp,connected_gnb,ue0_BS_ketnoi,ue0_handover,ue0_handover_to_type,Type
0,-85,-95,-110,-120,-130,000008,000008,0,000008,Xn
1,-84,-94,-109,-119,-129,000008,000008,0,000008,Xn
13,-72,-82,-97,-107,-117,000008,000008,1,000009,Xn
14,-71,-81,-96,-106,-116,000009,000009,0,000009,Xn
23,-130,-95,-140,-85,-140,000009,000009,1,000008,Xn
24,-130,-100,-140,-80,-140,000008,000008,0,000008,Xn
34,-60,-140,-70,-70,-70,000008,000008,1,000009,Xn
35,-55,-140,-65,-75,-65,000009,000009,0,000009,Xn
45,-105,-65,-85,-125,-85,000009,000009,1,000008,Xn
46,-110,-60,-90,-130,-90,000008,000008,0,000008,Xn
50,-130,-80,-110,-140,-110,000008,000008,0,000008,Xn
`
	err := os.WriteFile(testCSV, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}

	cfg := MeasurementCsvConfig{
		FilePath:    testCSV,
		StepDelayMs: 100,
	}

	steps, err := LoadMeasurementEventsFromCSV(cfg)
	if err != nil {
		t.Fatalf("LoadMeasurementEventsFromCSV() error = %v", err)
	}

	if len(steps) != 11 {
		t.Fatalf("Expected 11 steps, got %d", len(steps))
	}

	hoCount := 0
	for _, s := range steps {
		if s.HoTrigger == 1 {
			hoCount++
		}
	}
	if hoCount != 4 {
		t.Errorf("Expected 4 handover events, got %d", hoCount)
	}

	expectedHO := []struct {
		timestamp int
		from      string
		to        string
	}{
		{14, "000008", "000009"},
		{24, "000009", "000008"},
		{35, "000008", "000009"},
		{46, "000009", "000008"},
	}

	hoIdx := 0
	for _, s := range steps {
		if s.HoTrigger != 1 {
			continue
		}
		if hoIdx >= len(expectedHO) {
			t.Errorf("Unexpected extra handover at timestamp %d", s.Timestamp)
			continue
		}
		exp := expectedHO[hoIdx]
		if s.Timestamp != exp.timestamp {
			t.Errorf("HO %d: timestamp = %d, want %d", hoIdx, s.Timestamp, exp.timestamp)
		}
		if s.FromGnbId != exp.from {
			t.Errorf("HO %d: FromGnbId = %s, want %s", hoIdx, s.FromGnbId, exp.from)
		}
		if s.ToGnbId != exp.to {
			t.Errorf("HO %d: ToGnbId = %s, want %s", hoIdx, s.ToGnbId, exp.to)
		}
		hoIdx++
	}
}

func TestLoadMeasurementEventsFromCSV_MissingRequiredColumns(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		csv     string
		wantErr string
	}{
		{
			name:    "missing connected_gnb",
			csv:     "Bước,gnb1_rsrp\n0,-85\n",
			wantErr: "connected_gnb",
		},
		{
			name:    "missing Bước",
			csv:     "connected_gnb,gnb1_rsrp\n000008,-85\n",
			wantErr: "Bước",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCSV := filepath.Join(tmpDir, tt.name+".csv")
			err := os.WriteFile(testCSV, []byte(tt.csv), 0644)
			if err != nil {
				t.Fatalf("Failed to create test CSV: %v", err)
			}

			cfg := MeasurementCsvConfig{FilePath: testCSV}
			_, err = LoadMeasurementEventsFromCSV(cfg)
			if err == nil {
				t.Error("Expected error for missing column")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLoadMeasurementEventsFromCSV_TypeColumn(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("missing Type column defaults to Xn", func(t *testing.T) {
		testCSV := filepath.Join(tmpDir, "no_type.csv")
		csvContent := "Bước,connected_gnb\n0,000008\n1,000009\n"
		err := os.WriteFile(testCSV, []byte(csvContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test CSV: %v", err)
		}

		cfg := MeasurementCsvConfig{FilePath: testCSV}
		steps, err := LoadMeasurementEventsFromCSV(cfg)
		if err != nil {
			t.Fatalf("LoadMeasurementEventsFromCSV() error = %v", err)
		}

		for _, s := range steps {
			if s.HandoverType != "Xn" {
				t.Errorf("Step %d: HandoverType = %s, want Xn (default)", s.Timestamp, s.HandoverType)
			}
		}
	})

	t.Run("mixed Type values parsed correctly", func(t *testing.T) {
		testCSV := filepath.Join(tmpDir, "mixed_type.csv")
		csvContent := "Bước,connected_gnb,Type\n0,000008,Xn\n1,000009,N2\n2,000008,Xn\n"
		err := os.WriteFile(testCSV, []byte(csvContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test CSV: %v", err)
		}

		cfg := MeasurementCsvConfig{FilePath: testCSV}
		steps, err := LoadMeasurementEventsFromCSV(cfg)
		if err != nil {
			t.Fatalf("LoadMeasurementEventsFromCSV() error = %v", err)
		}

		if steps[0].HandoverType != "Xn" {
			t.Errorf("Step 0: HandoverType = %s, want Xn", steps[0].HandoverType)
		}
		if steps[1].HandoverType != "N2" {
			t.Errorf("Step 1: HandoverType = %s, want N2", steps[1].HandoverType)
		}
		if steps[2].HandoverType != "Xn" {
			t.Errorf("Step 2: HandoverType = %s, want Xn", steps[2].HandoverType)
		}
	})
}
