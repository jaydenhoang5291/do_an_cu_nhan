package scenarios

import (
	"context"
	"fmt"
	"maps"
	"os"
	"os/signal"
	"stormsim/internal/common/logger"
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/core/monitor"
	"stormsim/internal/core/uecontext"
	"stormsim/internal/core/uecontext/timer"
	"stormsim/pkg/config"
	"stormsim/pkg/model"
	"sync"
	"syscall"
	"time"
)

func durationToMs(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
}

func TestMeasurementHO(cfg *config.Config, csvCfg config.MeasurementCsvConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigStop := make(chan os.Signal, 1)
	signal.Notify(sigStop, os.Interrupt, syscall.SIGTERM)

	InitScenarioLogger(cfg, 50, 10, nil, ctx)

	testLogger.Info("=================== Measurement-based Handover Test ==================")

	measurementSteps, err := config.LoadMeasurementEventsFromCSV(csvCfg)
	if err != nil {
		testLogger.Fatal("Failed to load measurement data: %v", err)
		return
	}

	var wg sync.WaitGroup
	gnbs := createGnbs(len(cfg.GNodeBConfig.ListGnbs), cfg.GNodeBConfig, cfg.AMFs, cfg.Logging.GnbLogBufferSize, &wg, ctx)

	gnbMap := make(map[string]*gnbcontext.GnbContext)
	maps.Copy(gnbMap, gnbs)

	for _, gnb := range gnbMap {
		gnb.SetXnSimParams(csvCfg.XnPacketLoss, csvCfg.XnLatencyMs, csvCfg.XnJitterMs)
	}

	firstConnectedGnb := ""
	for _, step := range measurementSteps {
		if step.ConnectedGnb != "" {
			firstConnectedGnb = step.ConnectedGnb
			break
		}
	}
	if firstConnectedGnb == "" {
		testLogger.Fatal("No connected_gnb found in CSV")
		return
	}

	initialGnb, ok := gnbMap[firstConnectedGnb]
	if !ok {
		testLogger.Fatal("gNB %s from CSV not found in gnbMap", firstConnectedGnb)
		return
	}
	testLogger.Info("Using gNB %s as initial gNB (from CSV)", firstConnectedGnb)

	if csvCfg.T304DurationMs > 0 {
		timer.T304_duration = time.Duration(csvCfg.T304DurationMs) * time.Millisecond
		testLogger.Info("T304 duration set to %v (from config)", timer.T304_duration)
	} else {
		testLogger.Info("T304 duration using default %v", timer.T304_duration)
	}

	ueCtx := uecontext.CreateUe(cfg.DefaultUe, cfg.Logging.UeLogBufferSize, 0, initialGnb.GetId(), false, false, &wg, ctx)
	ueTasks := ueCtx.GetEventQueue()
	tracker := NewHandoverTracker()
	entries := make([]HoPhaseCSVEntry, 0)
	timerEvents := logger.NewTimerEventRingBuffer(10000)

	type timerTrialInfo struct {
		trialId   int
		hoType    string
		sourceGnb string
		targetGnb string
	}

	var currentTrialId int
	currentTrialInfo := timerTrialInfo{hoType: "Xn"}
	activeTimerInfo := make(map[string]timerTrialInfo)
	activeTimerStarts := make(map[string]time.Time)
	var currentTrialMu sync.Mutex

	timerHook := func(timerType timer.TimerType, prUeId int64, event string, elapsed time.Duration) {
		timerKey := fmt.Sprintf("%s-%d", timerType.String(), prUeId)

		currentTrialMu.Lock()
		if event == "start" {
			info := currentTrialInfo
			info.trialId = currentTrialId
			activeTimerInfo[timerKey] = info
			activeTimerStarts[timerKey] = time.Now()
		}
		info, ok := activeTimerInfo[timerKey]
		if !ok {
			info = currentTrialInfo
			info.trialId = currentTrialId
		}
		startTs := activeTimerStarts[timerKey]
		stopTs := time.Time{}
		if event == "stop" || event == "timeout" || event == "cancel" || event == "restart" {
			stopTs = time.Now()
			delete(activeTimerInfo, timerKey)
			delete(activeTimerStarts, timerKey)
		}
		currentTrialMu.Unlock()

		result := ""
		if event == "stop" {
			result = "success"
		} else if event == "timeout" {
			result = "timeout"
		}

		stopReason := ""
		if event == "stop" {
			stopReason = "success"
		} else if event == "timeout" {
			stopReason = "timeout"
		} else if event == "zombie" {
			stopReason = "zombie"
		} else if event == "cancel" {
			stopReason = "cancel"
			result = "cancel"
		} else if event == "transport_fail" {
			stopReason = "xn_transport_fail"
			result = "cancel"
		} else if event == "restart" {
			stopReason = "restart"
			result = "restart"
		}

		timerEvents.PushTimerEvent(logger.TimerEventEntry{
			Ts:         time.Now(),
			StartTs:    startTs,
			StopTs:     stopTs,
			TrialId:    info.trialId,
			Pdr:        csvCfg.PacketLoss,
			DelayMs:    float64(csvCfg.LatencyMs),
			HoType:     info.hoType,
			TimerName:  timerType.String(),
			Event:      event,
			ElapsedMs:  durationToMs(elapsed),
			Result:     result,
			StopReason: stopReason,
			SourceGnb:  info.sourceGnb,
			TargetGnb:  info.targetGnb,
		})
	}

	logHandoverSummary := func(trialId int, hoType, sourceGnb, targetGnb, result, failReason string, start, end time.Time) {
		total := end.Sub(start)
		timerEvents.PushSummaryEvent(logger.HandoverTrialSummaryEntry{
			Ts:            end,
			StartTs:       start,
			TrialId:       trialId,
			Pdr:           csvCfg.PacketLoss,
			DelayMs:       float64(csvCfg.LatencyMs),
			HoType:        hoType,
			OverallResult: result,
			FailReason:    failReason,
			TotalHoMs:     durationToMs(total),
			SourceGnb:     sourceGnb,
			TargetGnb:     targetGnb,
		})
	}

	ueCtx.GetTimerEngine().SetEventHook(timerHook)
	for _, gnb := range gnbMap {
		gnb.GetTimerEngine().SetEventHook(timerHook)
		// start watchdog on each gNB timer engine to detect zombie timers (diagnostic)
		gnb.GetTimerEngine().StartWatchdog(1*time.Second, 3.0)
	}

	// start watchdog on UE timer engine too
	ueCtx.GetTimerEngine().StartWatchdog(1*time.Second, 3.0)

	go func() {
		<-sigStop
		fmt.Println("\n[INTERRUPT] Stopping simulation safely...")
		testLogger.Warn("Received SIGINT/SIGTERM, closing...")
		cancel()

		time.Sleep(1 * time.Second)
		fmt.Println(tracker.PrintSummary())
		printMeasurementRlinkStats(ueCtx, gnbMap)
		csvContent := buildHoPhaseCSV(entries)
		if err := os.WriteFile("ho-time-log.csv", []byte(csvContent), 0644); err != nil {
			testLogger.Error("Failed to write HO phase CSV: %v", err)
		}

		timerEventList := timerEvents.GetTimerEvents()
		summaryEventList := timerEvents.GetSummaryEvents()
		exporter := logger.NewCSVExporter(".")
		if len(timerEventList) > 0 {
			csvFilename := fmt.Sprintf("timer-events-%d.csv", time.Now().Unix())
			if err := exporter.ExportTimerEventsCSV(timerEventList, csvFilename); err != nil {
				testLogger.Error("Failed to export timer events CSV: %v", err)
			} else {
				testLogger.Info("Exported %d timer events to %s (interrupt)", len(timerEventList), csvFilename)
			}
			if err := exporter.ExportTimerEventSummaryCSV(timerEventList, summaryEventList, csvFilename); err != nil {
				testLogger.Error("Failed to export timer summary CSV: %v", err)
			} else {
				testLogger.Info("Exported timer summary for %s (interrupt)", csvFilename)
			}
		}
		if len(summaryEventList) > 0 {
			summaryFilename := fmt.Sprintf("handover-trial-summary-%d.csv", time.Now().Unix())
			if err := exporter.ExportHandoverTrialSummariesCSV(summaryEventList, summaryFilename); err != nil {
				testLogger.Error("Failed to export handover trial summary CSV: %v", err)
			} else {
				testLogger.Info("Exported %d handover trial summaries to %s (interrupt)", len(summaryEventList), summaryFilename)
			}
		}

		os.Exit(0)
	}()

	fmt.Println("##################################################")
	fmt.Println("STORM-SIM MEASUREMENT ENGINE: VERSION 3.0 (STABLE)")
	fmt.Println("##################################################")

	// Temporarily disable loss for stable registration
	originalLoss := cfg.Simulation.PacketLoss
	cfg.Simulation.PacketLoss = 0
	for _, gnb := range gnbMap {
		gnb.SetRadioParams(0, 0, 0)
		gnb.SetXnSimParams(0, 0, 0)
	}
	testLogger.Info("Temporary 0%% loss enabled for stable Registration/PDU setup...")

	testLogger.Info("Step 0: Registering UE...")
	ueTasks.AssignTask(&uecontext.EventUeData{EventType: model.RegisterInit})

	// Wait for registration success (max 30s)
	regTimeout := time.After(30 * time.Second)
	regTicker := time.NewTicker(1 * time.Second)
	defer regTicker.Stop()

	registered := false
	for !registered {
		select {
		case <-regTicker.C:
			currState := ueCtx.GetMMState().CurrentState()
			if currState == model.Registered {
				registered = true
				testLogger.Info("UE Registered successfully.")
			} else {
				testLogger.Info("Waiting for Registration... (Current State: %v)", currState)
			}
		case <-regTimeout:
			testLogger.Fatal("UE failed to register even with 0%% loss.")
			return
		}
	}

	testLogger.Info("Step 1: Establishing PDU session...")
	ueTasks.AssignTask(&uecontext.EventUeData{EventType: model.PduSessionInit})

	// Wait for PDU session success (max 20s)
	pduTimeout := time.After(20 * time.Second)
	pduTicker := time.NewTicker(1 * time.Second)
	defer pduTicker.Stop()

	pduActive := false
	for !pduActive {
		select {
		case <-pduTicker.C:
			smState := ueCtx.GetSMState()
			if smState != nil {
				currState := smState.CurrentState()
				if currState == model.PDUSessionActive {
					pduActive = true
					testLogger.Info("PDU Session Active. Restoring configured Loss: %.2f", originalLoss)
				} else {
					testLogger.Info("Waiting for PDU Session... (Current State: %v)", currState)
				}
			}
		case <-pduTimeout:
			testLogger.Fatal("UE failed to establish PDU session even with 0%% loss.")
			return
		}
	}

	// Restore original loss for handover testing
	cfg.Simulation.PacketLoss = originalLoss
	for _, gnb := range gnbMap {
		gnb.SetRadioParams(originalLoss, csvCfg.LatencyMs, csvCfg.JitterMs)
		gnb.SetXnSimParams(csvCfg.XnPacketLoss, csvCfg.XnLatencyMs, csvCfg.XnJitterMs)
	}
	time.Sleep(1 * time.Second)

	currentGnbId := initialGnb.GetId()
	testLogger.Info("Starting measurement-based handover sequence...")

	stepDelay := time.Duration(csvCfg.StepDelayMs) * time.Millisecond
	if stepDelay == 0 {
		stepDelay = 1 * time.Second
	}

	for i, step := range measurementSteps {
		currentTrialMu.Lock()
		currentTrialId = i
		currentTrialMu.Unlock()

		select {
		case <-ctx.Done():
			return
		default:
		}

		if csvCfg.ChoMode && len(step.RsrpValues) > 0 {
			ueCtx.UpdateRsrpValues(step.RsrpValues)
		}

		fmt.Printf("\n[%3d] timestamp=%ds connected=%s ho_trigger=%d\n", i, step.Timestamp, currentGnbId, step.HoTrigger)

		if step.HoTrigger == 1 {
			targetGnbId := step.ToGnbId
			targetGnb, ok := gnbMap[targetGnbId]

			if !ok || currentGnbId == targetGnbId {
				testLogger.Warn("Handover skipped: Target invalid or same as current")
				time.Sleep(stepDelay)
				continue
			}

			hoTypeForLog := step.HandoverType
			if hoTypeForLog == "" {
				hoTypeForLog = "Xn"
			}
			currentTrialMu.Lock()
			currentTrialInfo = timerTrialInfo{
				trialId:   i,
				hoType:    hoTypeForLog,
				sourceGnb: currentGnbId,
				targetGnb: targetGnbId,
			}
			currentTrialMu.Unlock()

			// Global Inventory Check & Cleanup: Ensure UE only exists in current GNB
			gnbcontext.Gnbs.Range(func(key, value any) bool {
				g := value.(*gnbcontext.GnbContext)
				if g.GetId() == currentGnbId {
					return true
				}
				// If found in other GNB, reset it
				if _, err := g.GetGnbUeByPrUeId(ueCtx.GetId()); err == nil {
					testLogger.Warn("Cleaning up stale context for UE %d in gNB %s", ueCtx.GetId(), g.GetId())
					g.ResetUeContext(ueCtx.GetId())
				}
				return true
			})

			// Clean up target gNB context before triggering handover
			targetGnb.ResetUeContext(ueCtx.GetId())

			hoSuccess := false
			if csvCfg.ChoMode {
				testLogger.Info("[Step %d] CHO Mode: preparing candidates for UE %d from %s", step.Timestamp, ueCtx.GetId(), currentGnbId)

				sourceGnb := gnbMap[currentGnbId]
				if sourceGnb == nil {
					testLogger.Warn("Source gNB %s not found, skipping", currentGnbId)
					time.Sleep(stepDelay)
					continue
				}

				candidates := make([]string, 0)
				for gnbId, rsrp := range step.RsrpValues {
					if gnbId != currentGnbId && rsrp > -120 {
						candidates = append(candidates, gnbId)
					}
				}
				if len(candidates) == 0 {
					candidates = append(candidates, targetGnbId)
				}

				tracker.StartHandover(step.Timestamp, currentGnbId, targetGnbId, 1)
				startTime := time.Now()

				sourceGnb.PrepareChoHandover(candidates, ueCtx.GetId())

				ueId := ueCtx.GetId()
				ticker := time.NewTicker(200 * time.Millisecond)
				hoTimeout := timer.T304_duration * 5 // 5x T304 for retries + margin
				if hoTimeout < 2*time.Second {
					hoTimeout = 2 * time.Second
				}
				timeout := time.After(hoTimeout)
				completed := false
				choSuccess := false

				for !completed {
					select {
					case <-ticker.C:
						if targetGnb.IsHandoverSuccess(ueId) {
							choSuccess = true
							completed = true
						}
					case <-timeout:
						completed = true
					case <-ctx.Done():
						ticker.Stop()
						return
					}
				}
				ticker.Stop()

				endTime := time.Now()
				if choSuccess {
					duration := endTime.Sub(startTime)
					tracker.CompleteHandoverWithTiming(step.Timestamp, true, "", endTime, duration)
					logHandoverSummary(i, hoTypeForLog, currentGnbId, targetGnbId, "success", "", startTime, endTime)
					testLogger.Info("[%3d] CHO SUCCESS: UE %d → %s (duration=%v)", step.Timestamp, ueId, targetGnbId, duration)
				} else {
					duration := endTime.Sub(startTime)
					tracker.CompleteHandoverWithTiming(step.Timestamp, false, "Timeout", endTime, duration)
					logHandoverSummary(i, hoTypeForLog, currentGnbId, targetGnbId, "fail", "Timeout", startTime, endTime)
					testLogger.Warn("[%3d] CHO TIMEOUT: UE %d did not complete to %s", step.Timestamp, ueId, targetGnbId)
				}
				hoSuccess = choSuccess
			} else {
				hoTypeLabel := hoTypeForLog
				testLogger.Info("[Step %d] %s Handover from %s to %s", step.Timestamp, hoTypeLabel, currentGnbId, targetGnbId)

				hoType := 1
				if hoTypeLabel == "N2" {
					hoType = 2
				}

				xnLoss := csvCfg.XnPacketLoss
				if xnLoss == 0 {
					xnLoss = csvCfg.PacketLoss
				}
				func() {
					hoProc := monitor.NewHoProcedure(
						ueCtx.GetId(),
						currentGnbId,
						targetGnbId,
						hoType,
						xnLoss,
						time.Duration(csvCfg.LatencyMs)*time.Millisecond,
						time.Duration(csvCfg.JitterMs)*time.Millisecond,
					)

					prevHoHandler := gnbcontext.HoEventHandler
					gnbcontext.HoEventHandler = func(prUeId int64, eventType model.EventType, data interface{}) {
						if prUeId != ueCtx.GetId() {
							if prevHoHandler != nil {
								prevHoHandler(prUeId, eventType, data)
							}
							return
						}
						switch eventType {
						case model.HoRlinkPrepareReqSentEvent, model.HoPathSwitchRequestEvent:
							hoProc.SendEvent(eventType, nil)
						default:
							if prevHoHandler != nil {
								prevHoHandler(prUeId, eventType, data)
							}
						}
					}
					defer func() {
						gnbcontext.HoEventHandler = prevHoHandler
					}()

					tracker.StartHandover(step.Timestamp, currentGnbId, targetGnbId, hoType)

					hoProc.StartHo()

					failReason := "Timeout"
					if csvCfg.FailMode {
						failReason = "Timer expiry (loss=100%)"
					}

					ueId := ueCtx.GetId()
					ticker := time.NewTicker(200 * time.Millisecond)
					defer ticker.Stop()
					hoTimeout := timer.T304_duration * 5 // 5x T304 for retries + margin
					if hoTimeout < 2*time.Second {
						hoTimeout = 2 * time.Second
					}
					timeout := time.After(hoTimeout)
					completed := false

					for !completed {
						select {
						case <-ticker.C:
							if targetGnb.IsHandoverSuccess(ueId) {
								endTime := time.Now()
								duration := endTime.Sub(hoProc.GetStartTime())
								hoProc.SendEvent(model.HoRlinkSetupPduSessonEvent, nil)
								tracker.CompleteHandoverWithTiming(step.Timestamp, true, "", endTime, duration)
								logHandoverSummary(i, hoTypeForLog, currentGnbId, targetGnbId, "success", "", hoProc.GetStartTime(), endTime)
								testLogger.Info("[%3d] Handover SUCCESS (duration=%v)", step.Timestamp, duration)
								entries = append(entries, HoPhaseCSVEntry{Step: step.Timestamp, Proc: hoProc})
								hoSuccess = true
								completed = true
							}
						case <-timeout:
							endTime := time.Now()
							duration := endTime.Sub(hoProc.GetStartTime())
							hoProc.SendEvent(model.HoFailEvent, &monitor.HoEventData{FailureReason: failReason})
							tracker.CompleteHandoverWithTiming(step.Timestamp, false, failReason, endTime, duration)
							logHandoverSummary(i, hoTypeForLog, currentGnbId, targetGnbId, "fail", failReason, hoProc.GetStartTime(), endTime)
							if csvCfg.FailMode {
								testLogger.Warn("[%3d] Handover FAILED - timer expired (expected in fail mode, duration=%v)", step.Timestamp, duration)
							} else {
								testLogger.Warn("[%3d] Handover FAILED - timeout (duration=%v)", step.Timestamp, duration)
							}
							entries = append(entries, HoPhaseCSVEntry{Step: step.Timestamp, Proc: hoProc})
							completed = true
						case <-ctx.Done():
							return
						}
					}
				}()
			}

			if hoSuccess {
				currentGnbId = targetGnbId
			}
		}
		time.Sleep(stepDelay)
	}

	testLogger.Info("Sequence completed.")
	fmt.Println(tracker.PrintSummary())
	printMeasurementRlinkStats(ueCtx, gnbMap)
	csvContent := buildHoPhaseCSV(entries)
	if err := os.WriteFile("ho-time-log.csv", []byte(csvContent), 0644); err != nil {
		testLogger.Error("Failed to write HO phase CSV: %v", err)
	}

	timerEventList := timerEvents.GetTimerEvents()
	summaryEventList := timerEvents.GetSummaryEvents()
	exporter := logger.NewCSVExporter(".")
	if len(timerEventList) > 0 {
		csvFilename := fmt.Sprintf("timer-events-%d.csv", time.Now().Unix())
		if err := exporter.ExportTimerEventsCSV(timerEventList, csvFilename); err != nil {
			testLogger.Error("Failed to export timer events CSV: %v", err)
		} else {
			testLogger.Info("Exported %d timer events to %s", len(timerEventList), csvFilename)
		}
		if err := exporter.ExportTimerEventSummaryCSV(timerEventList, summaryEventList, csvFilename); err != nil {
			testLogger.Error("Failed to export timer summary CSV: %v", err)
		} else {
			testLogger.Info("Exported timer summary for %s", csvFilename)
		}
	}
	if len(summaryEventList) > 0 {
		summaryFilename := fmt.Sprintf("handover-trial-summary-%d.csv", time.Now().Unix())
		if err := exporter.ExportHandoverTrialSummariesCSV(summaryEventList, summaryFilename); err != nil {
			testLogger.Error("Failed to export handover trial summary CSV: %v", err)
		} else {
			testLogger.Info("Exported %d handover trial summaries to %s", len(summaryEventList), summaryFilename)
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		testLogger.Info("All components stopped.")
	case <-time.After(2 * time.Second):
		testLogger.Warn("Timeout waiting for components, forcing exit.")
	}
}
