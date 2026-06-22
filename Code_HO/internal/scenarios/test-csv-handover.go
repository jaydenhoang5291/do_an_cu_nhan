package scenarios

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/core/uecontext"
	"stormsim/pkg/config"
	"stormsim/pkg/model"
	"sync"
	"syscall"
	"time"
)

// TestCSVHandover runs handover scenario based on CSV data
func TestCSVHandover(cfg *config.Config, csvCfg config.CSVHandoverConfig) {
	fmt.Println("==================================================")
	fmt.Println("--- STORM-SIM NEW ENGINE V2.0 LOADED ---")
	fmt.Println("==================================================")

	var wg sync.WaitGroup
	sigStop := make(chan os.Signal, 1)
	signal.Notify(sigStop, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	InitScenarioLogger(cfg, 50, 10, nil, ctx)

	safeLogInfo := func(format string, v ...interface{}) {
		if testLogger != nil {
			testLogger.Info(format, v...)
		} else {
			fmt.Printf("[INFO] "+format+"\n", v...)
		}
	}
	safeLogWarn := func(format string, v ...interface{}) {
		if testLogger != nil {
			testLogger.Warn(format, v...)
		} else {
			fmt.Printf("[WARN] "+format+"\n", v...)
		}
	}
	safeLogError := func(format string, v ...interface{}) {
		if testLogger != nil {
			testLogger.Error(format, v...)
		} else {
			fmt.Printf("[ERROR] "+format+"\n", v...)
		}
	}
	safeLogFatal := func(format string, v ...interface{}) {
		if testLogger != nil {
			testLogger.Fatal(format, v...)
		} else {
			fmt.Printf("[FATAL] "+format+"\n", v...)
			os.Exit(1)
		}
	}

	fmt.Println("=================== CSV Handover Test ==================")
	fmt.Printf("Loading handover events from: %s\n", csvCfg.FilePath)

	handoverSteps, err := config.LoadHandoverEventsFromCSV(csvCfg)
	if err != nil {
		fmt.Printf("\n[FATAL] Lỗi đọc file CSV: %v\n", err)
		fmt.Println("Vui lòng kiểm tra lại đường dẫn hoặc định dạng file CSV.")
		return
	}

	safeLogInfo("Loaded %d handover events from CSV", len(handoverSteps))

	if len(handoverSteps) == 0 {
		safeLogWarn("No handover events found in CSV file")
		return
	}

	gnbs := createGnbs(len(cfg.GNodeBConfig.ListGnbs), cfg.GNodeBConfig, cfg.AMFs, cfg.Logging.GnbLogBufferSize, &wg, ctx)

	var initialGnb *gnbcontext.GnbContext
	gnbMap := make(map[string]*gnbcontext.GnbContext)
	for id, gnb := range gnbs {
		if initialGnb == nil {
			initialGnb = gnb
		}
		gnbMap[id] = gnb
	}

	if initialGnb == nil {
		fmt.Println("[FATAL] No gNB available for initial connection. Check your config.yml")
		return
	}

	safeLogInfo("Creating UE with initial gNB: %s", initialGnb.GetId())
	ueCtx := uecontext.CreateUe(
		cfg.DefaultUe,
		cfg.Logging.UeLogBufferSize,
		0,
		initialGnb.GetId(),
		false,
		false,
		&wg,
		ctx,
	)
	ueTasks := ueCtx.GetEventQueue()

	tracker := NewHandoverTracker()

	// Temporarily disable loss for stable registration
	originalLoss := cfg.Simulation.PacketLoss
	cfg.Simulation.PacketLoss = 0
	for _, gnb := range gnbMap {
		gnb.SetRadioLoss(0)
		gnb.SetXnSimParams(0, 0, 0)
	}
	safeLogInfo("Temporary 0%% loss enabled for stable Registration/PDU setup...")

	safeLogInfo("Step 0: Registering UE...")
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
				safeLogInfo("UE Registered successfully.")
			} else {
				safeLogInfo("Waiting for Registration... (Current State: %v)", currState)
			}
		case <-regTimeout:
			safeLogFatal("UE failed to register even with 0%% loss. Check AMF/gNB connectivity.")
			return
		}
	}

	safeLogInfo("Step 1: Establishing PDU session...")
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
					safeLogInfo("PDU Session Active. Restoring configured Loss: %.2f", originalLoss)
				} else {
					safeLogInfo("Waiting for PDU Session... (Current State: %v)", currState)
				}
			}
		case <-pduTimeout:
			safeLogFatal("UE failed to establish PDU session even with 0%% loss.")
			return
		}
	}

	// Restore original loss for handover testing
	cfg.Simulation.PacketLoss = originalLoss
	for _, gnb := range gnbMap {
		gnb.SetRadioLoss(originalLoss)
		gnb.SetXnSimParams(csvCfg.XnPacketLoss, csvCfg.XnLatencyMs, csvCfg.XnJitterMs)
	}
	time.Sleep(1 * time.Second)

	currentGnb := initialGnb

	go func() {
		<-sigStop
		fmt.Println("\n" + tracker.PrintSummary())
		printDetailedResults(tracker)
		cancel()
		os.Exit(0)
	}()

	safeLogInfo("Starting CSV handover sequence...")
	stepDelay := time.Duration(csvCfg.StepDelayMs) * time.Millisecond
	if stepDelay == 0 {
		stepDelay = 2 * time.Second
	}

	for _, step := range handoverSteps {
		select {
		case <-ctx.Done():
			return
		default:
		}

		targetGnb, ok := gnbMap[step.ToGnbId]
		if !ok {
			safeLogError("Step %d: Target gNB %s not found in config, skipping", step.Step, step.ToGnbId)
			continue
		}

		if currentGnb.GetId() == targetGnb.GetId() {
			safeLogWarn("Step %d: Source and target gNB are same (%s), skipping", step.Step, step.ToGnbId)
			continue
		}

		hoTypeStr := "Xn"
		if step.IsN2Handover() {
			hoTypeStr = "N2"
		}
		safeLogInfo("Step %d: %s Handover from %s to %s", step.Step, hoTypeStr, currentGnb.GetId(), targetGnb.GetId())

		tracker.StartHandover(step.Step, currentGnb.GetId(), targetGnb.GetId(), step.HandoverType)

		// Global Inventory Check & Cleanup: Ensure UE only exists in current GNB
		gnbcontext.Gnbs.Range(func(key, value any) bool {
			g := value.(*gnbcontext.GnbContext)
			if g.GetId() == currentGnb.GetId() {
				return true
			}
			// If found in other GNB, reset it
			if _, err := g.GetGnbUeByPrUeId(ueCtx.GetId()); err == nil {
				safeLogWarn("Cleaning up stale context for UE %d in gNB %s", ueCtx.GetId(), g.GetId())
				g.ResetUeContext(ueCtx.GetId())
			}
			return true
		})

		targetGnb.ResetUeContext(ueCtx.GetId())

		if triggerCSVHandover(currentGnb, targetGnb, ueCtx, tracker, step.Step, step.IsXnHandover(), csvCfg) {
			currentGnb = targetGnb
		}

		time.Sleep(stepDelay)
	}

	safeLogInfo("CSV handover sequence completed")
	fmt.Println(tracker.PrintSummary())
	printDetailedResults(tracker)

	wg.Wait()
}

func triggerCSVHandover(
	oldGnb, newGnb *gnbcontext.GnbContext,
	ueCtx *uecontext.UeContext,
	tracker *HandoverTracker,
	step int,
	isXn bool,
	cfg config.CSVHandoverConfig,
) bool {
	ueId := ueCtx.GetId()

	if isXn {
		gnbcontext.TriggerXnHandover(oldGnb, newGnb, ueId, cfg.XnPacketLoss, time.Duration(cfg.XnLatencyMs)*time.Millisecond, time.Duration(cfg.XnJitterMs)*time.Millisecond)
	} else {
		gnbcontext.TriggerNgapHandover(oldGnb, newGnb, ueId, cfg.PacketLoss, 0, 0)
	}

	hoTypeStr := "Xn"
	if !isXn {
		hoTypeStr = "N2"
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	timeout := time.After(15 * time.Second) // covers all timer durations (max = TNGRELOCoverall 10s)
	for {
		select {
		case <-ticker.C:
			if newGnb.IsHandoverSuccess(ueId) {
				if ueCtx.GetMMState().CurrentState() == model.Registered {
					tracker.CompleteHandover(step, true, "")
					testLogger.Info("Step %d: %s Handover SUCCESS", step, hoTypeStr)
					return true
				} else {
					tracker.CompleteHandover(step, false, "UE not in Registered state")
					testLogger.Warn("Step %d: %s Handover FAILED - UE not registered", step, hoTypeStr)
					return false
				}
			}
		case <-timeout:
			ticker.Stop()
			if cfg.FailMode {
				tracker.CompleteHandover(step, false, "Timer expiry (loss=100%)")
				testLogger.Warn("Step %d: %s Handover FAILED - timer expired (expected in fail mode)", step, hoTypeStr)
			} else {
				tracker.CompleteHandover(step, false, "Timeout")
				testLogger.Warn("Step %d: %s Handover FAILED - timeout", step, hoTypeStr)
			}
			return false
		}
	}
}

// printDetailedResults prints detailed handover results
func printDetailedResults(tracker *HandoverTracker) {
	results := tracker.GetResults()
	if len(results) == 0 {
		return
	}

	fmt.Println("\n========== Detailed Handover Results ==========")
	fmt.Printf("%-6s %-10s %-10s %-6s %-10s %-12s %s\n",
		"Step", "From", "To", "Type", "Status", "Duration", "Reason")
	fmt.Println("---------------------------------------------------------------")

	for _, r := range results {
		hoType := "Xn"
		if r.HandoverType == 2 {
			hoType = "N2"
		}
		status := "SUCCESS"
		if !r.Success {
			status = "FAILED"
		}
		fmt.Printf("%-6d %-10s %-10s %-6s %-10s %-12s %s\n",
			r.Step, r.FromGnb, r.ToGnb, hoType, status, r.Duration, r.FailureReason)
	}
	fmt.Println("================================================")
}

func safeLogInfo(format string, v ...interface{}) {
	if testLogger != nil {
		testLogger.Info(format, v...)
	} else {
		fmt.Printf("INFO "+format+"\n", v...)
	}
}

func safeLogWarn(format string, v ...interface{}) {
	if testLogger != nil {
		testLogger.Warn(format, v...)
	} else {
		fmt.Printf("WARN "+format+"\n", v...)
	}
}

func safeLogFatal(format string, v ...interface{}) {
	if testLogger != nil {
		testLogger.Fatal(format, v...)
	} else {
		fmt.Printf("FATAL "+format+"\n", v...)
		os.Exit(1)
	}
}
