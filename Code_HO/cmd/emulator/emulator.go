package main

import (
	"fmt"
	"os"
	"stormsim/internal/common/logger"
	"stormsim/internal/scenarios"
	"stormsim/monitoring"
	"stormsim/pkg/config"

	"github.com/urfave/cli/v2"
)

const version = "1.0.1"

var log *logger.Logger

func init() {
	log = logger.InitLogger("info", nil)
}

func main() {
	app := &cli.App{
		Name:    "stormsim",
		Version: version,
		Usage:   "5G Mobile Network Simulator",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
			},
			&cli.PathFlag{
				Name:  "pcap",
				Usage: "Capture traffic to given PCAP file when a path is given",
			},
			&cli.PathFlag{
				Name:    "replay",
				Aliases: []string{"r"},
				Usage:   "Load replay file for reproducing recorded scenarios (works with 1 UE only)",
			},
			&cli.BoolFlag{
				Name:    "config-help",
				Aliases: []string{"ch"},
				Usage:   "Show detailed configuration help",
			},
			&cli.PathFlag{
				Name:  "csv",
				Usage: "Load handover events from CSV file for mobility simulation",
			},
			&cli.PathFlag{
				Name:  "csv-measurement",
				Usage: "Load measurement-based handover events from CSV file",
			},
			&cli.StringFlag{
				Name:  "gnb-map",
				Usage: "gNB ID mapping from CSV to config (format: 'csvId1:gnbId1,csvId2:gnbId2')",
			},
			&cli.IntFlag{
				Name:  "step-delay",
				Usage: "Delay between handover steps in milliseconds (default: 1000)",
				Value: 1000,
			},
			&cli.BoolFlag{
				Name:  "fail-mode",
				Usage: "Enable failure mode: inject 100% packet loss to trigger handover timer expiry",
			},
			&cli.BoolFlag{
				Name:  "cho-mode",
				Usage: "Enable Conditional Handover (CHO) per 3GPP Rel-16 with UE-side condition evaluation",
			},
		},
		Action: func(c *cli.Context) error {
			// Check if config help is requested
			if c.Bool("config-help") {
				showConfigHelp()
				return nil
			}

			return runScenarios(c)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal("Failed to run app: %v", err)
	}
}

func runScenarios(c *cli.Context) error {
	cfg := setConfigFile(*c)

	log.Info("StormSim version %s", version)
	log.Info("---------------------------------------")
	log.Info("Starting 5G Network Simulation")
	log.Info("Default Control interface IP/Port: %s/%d", cfg.GNodeBConfig.DefaultControlIF.Ip, cfg.GNodeBConfig.DefaultControlIF.Port)
	log.Info("Default Data interface IP/Port: %s/%d", cfg.GNodeBConfig.DefaultDataIF.Ip, cfg.GNodeBConfig.DefaultDataIF.Port)
	for _, amf := range cfg.AMFs {
		log.Info("AMF IP/Port: %s/%d", amf.Ip, amf.Port)
	}
	log.Info("---------------------------------------")

	// Enable PCAP capture if specified
	if c.IsSet("pcap") {
		monitoring.CaptureTraffic(c.Path("pcap"))
	}

	failMode := c.Bool("fail-mode")
	if failMode {
		log.Warn("FAIL MODE enabled: handovers will use 100%% packet loss to trigger timer expiry")
	}

	choMode := c.Bool("cho-mode")
	if choMode {
		log.Info("CHO MODE enabled: Conditional Handover with UE-side condition evaluation")
	}

	// Check if CSV handover mode is enabled
	if c.IsSet("csv-measurement") {
		csvPath := c.Path("csv-measurement")
		stepDelay := c.Int("step-delay")

		log.Info("Running in CSV MEASUREMENT HANDOVER mode with file: %s", csvPath)

		csvCfg := config.MeasurementCsvConfig{
			FilePath:     csvPath,
			StepDelayMs:  stepDelay,
			PacketLoss:   cfg.Simulation.PacketLoss,
			LatencyMs:    cfg.Simulation.LatencyMs,
			JitterMs:     cfg.Simulation.JitterMs,
			XnPacketLoss: cfg.Simulation.XnPacketLoss,
			XnLatencyMs:  cfg.Simulation.XnLatencyMs,
			XnJitterMs:   cfg.Simulation.XnJitterMs,
			FailMode:     failMode,
			ChoMode:      choMode,
		}

		if failMode {
			csvCfg.PacketLoss = 1.0
		}

		if cfg.Simulation.Enable {
			log.Info("Network simulation enabled: loss=%.2f, delay=%dms, jitter=%dms",
				cfg.Simulation.PacketLoss, cfg.Simulation.LatencyMs, cfg.Simulation.JitterMs)
		}
		if cfg.Simulation.XnPacketLoss > 0 || cfg.Simulation.XnLatencyMs > 0 {
			log.Info("Xn simulation enabled: loss=%.2f, delay=%dms, jitter=%dms",
				cfg.Simulation.XnPacketLoss, cfg.Simulation.XnLatencyMs, cfg.Simulation.XnJitterMs)
		}

		scenarios.TestMeasurementHO(&cfg, csvCfg)
	} else if c.IsSet("csv") {
		csvPath := c.Path("csv")
		gnbMapStr := c.String("gnb-map")
		stepDelay := c.Int("step-delay")

		log.Info("Running in CSV HANDOVER mode with file: %s", csvPath)

		gnbMapping, err := config.ParseGnbMapping(gnbMapStr)
		if err != nil {
			log.Fatal("Invalid gnb-map format: %v", err)
			return err
		}

		csvCfg := config.CSVHandoverConfig{
			FilePath:     csvPath,
			GnbIdMapping: gnbMapping,
			StepDelayMs:  stepDelay,
			FailMode:     failMode,
			PacketLoss:   cfg.Simulation.PacketLoss,
			XnPacketLoss: cfg.Simulation.XnPacketLoss,
			XnLatencyMs:  cfg.Simulation.XnLatencyMs,
			XnJitterMs:   cfg.Simulation.XnJitterMs,
		}

		if failMode {
			csvCfg.PacketLoss = 1.0
			csvCfg.XnPacketLoss = 1.0
		}

		scenarios.TestCSVHandover(&cfg, csvCfg)
	} else if c.IsSet("replay") {
		// Check if replay mode is enabled
		replayFile := c.String("replay")
		log.Info("Running in REPLAY mode with file: %s", replayFile)
		scenarios.TestSingleUE(&cfg, true, replayFile)
	} else {
		// Run normal scenarios
		scenarios.TestScenarios(&cfg)
	}

	return nil
}

func setConfigFile(c cli.Context) (cfg config.Config) {
	configPath := c.String("config")
	cfg = config.LoadConfig(configPath)
	return
}

func showConfigHelp() {
	fmt.Println(`5G Mobile Network Simulator - Configuration Guide
=================================================

Usage: 
  sudo ./stormsim -c config/config.yml          # Run scenarios with custom config
  sudo ./stormsim                               # Run scenarios with default config
  sudo ./stormsim -r replay_file.log -c config/config.yml           # Replay recorded scenarios (1 UE only)
  sudo ./stormsim help                          # Show this help

Configuration Options in config.yml:
=====================================

1. NUMBER OF UEs AND gNBs:
   Configure in 'scenarios' section:
   scenarios:
     - nUEs: 5                    # Number of User Equipment to simulate
       gnbs: ["000008", "000009"] # List of gNodeB IDs to use
   
   Note: For handover events, you need more than 1 gNB configured!

2. ADDING/CUSTOMIZING gNBs:
   Add gNBs in 'gnodeb.listGnbs' section:
   gnodeb:
     listGnbs:
       - gnbid: "000008"          # Unique gNodeB ID
         tac: "000001"            # Tracking Area Code
         plmn:
           mcc: "208"             # Mobile Country Code
           mnc: "93"              # Mobile Network Code
         slicesupportlist:
           - sst: "01"            # Slice/Service Type
             sd: "010203"         # Slice Differentiator

3. CUSTOMIZING DEFAULT UE PROFILE:
   Modify 'defaultUe' section:
   defaultUe:
     msin: "0000000000"           # Mobile Subscriber ID Number
     key: "14b23ceb27e95eb..."    # Authentication key
     opc: "a3e3c63de23b66..."     # Operator key
     amf: "8000"                  # Authentication Management Field
     sqn: "00000000"              # Sequence Number
     dnn: "internet"              # Data Network Name

4. UE EVENTS CONFIGURATION:
   Add events to 'scenarios.ueEvents' section:
   scenarios:
     - ueEvents:
         - event: "RegisterInit Event"
           delay: 0
           register_type: 0       # 0: Initial, other values for different types
         - event: "PduSessionInit Event" 
           delay: 2               # Delay in seconds
         - event: "DeregistraterInit Event"
           delay: 5
           deregister_type: 0     # 0: not switch off, 1: switch off

   Available Events (from event-state.go):
   - RegisterInit Event           # Start registration process
   - DeregistraterInit Event      # Start deregistration  
   - ServiceRequestInit Event     # Service request procedure
   - PduSessionInit Event         # Establish PDU session
   - DestroyPduSession Event      # Release PDU session
   - XnHandover Event             # Xn interface handover
   - N2Handover Event             # N2 interface handover
   - Terminate Event              # Graceful UE termination
   - Kill Event                   # Force kill UE

5. RECORDING SIMULATOR (Fuzzy Testing):
   Enable recording in 'testconf' section:
   testconf:
     enableFuzz: true             # Enable fuzzy testing and recording
   
   Note: Recording only works with 1 UE for fuzzy testing scenarios

6. REPLAY FUNCTIONALITY:
   Use the -r flag to replay recorded scenarios:
   sudo ./stormsim -r path/to/replay_file.log -c config/config.yml
   
   Note: Replay mode only supports 1 UE scenarios

7. EXTERNAL CLIENT INTERACTION:
   Enable remote server in 'remote' section:
   remote:
     enable: true               # Enable remote API server
     ip: "0.0.0.0"             # Server bind IP (0.0.0.0 for all interfaces)
     port: 4000                # Server port
   
   This allows external clients to interact with the simulator via REST API

8. ADDITIONAL SETTINGS:
   - Control/Data interfaces: Set in 'gnodeb.controlif' and 'gnodeb.dataif'
   - AMF endpoints: Configure in 'amfif' section
   - Log level: Set in 'loglevel' (info, debug, trace, warn, error, fatal, panic)
   - Security settings: Configure encryption/integrity in 'defaultUe' section

=====================================
For more detailed examples, see the provided config.yml file.
If you get bug, please create new ISSUE or contact me at lvdund@gmail.com`)
}
