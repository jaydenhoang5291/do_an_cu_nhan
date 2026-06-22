# WORKER POOL AUTO-ADAPTATION CONFIGURATION

## Overview

The worker pool system has been enhanced with **automatic hardware detection** to adapt to different CPU/RAM configurations.

## Key Changes

### 1. Auto-Detection (Default Behavior)
When `maxPool=0` in `InitWorkerPool()`, the system automatically calculates optimal pool sizes based on:
- **CPU cores** (`runtime.NumCPU()`)
- **Workload** (number of UEs and gNodeBs)
- **Memory constraints**

### 2. Intelligent Subpool Allocation
The improved algorithm allocates workers as:
- **MM (Mobility Management)**: 40% of remaining pool (most frequent operations)
- **SM (Session Management)**: 35% of remaining pool
- **GNB**: 25% of remaining pool
- **SCTP**: ~16.7% of total pool (I/O bound, needs fewer workers)

### 3. Critical Bug Fixes
**FIXED**: Default configuration bug where `maxSctpNgapWorker = maxCapPool` resulted in **zero workers** for MM/SM/GNB subpools.

## Configuration Options

### Option 1: Auto-Detection (Recommended)
```go
// In test-with-custom-scenarios.go
InitScenarioLogger(cfg, 0, 0, httpSrv, ctx)
```
This automatically calculates optimal pool sizes based on:
- CPU cores: `numCPU * 1500` workers
- UE count: `nUEs * 3` workers minimum
- gNodeB count: `nGnbs * 100` workers overhead

**Example output on 8-core system with 1000 UEs:**
```
[Pool] Auto-detected: Total=12000, SCTP=2000, MM=4000, SM=3500, GNB=2500 (CPUs=8)
```

### Option 2: Manual Override
```go
// Custom pool sizes
InitScenarioLogger(cfg, 15000, 2500, httpSrv, ctx)
```
- `maxPool=15000`: Total worker capacity
- `nSctpWorker=2500`: SCTP workers
- System validates that SCTP < maxPool and auto-adjusts if needed

### Option 3: Hybrid (Manual Total, Auto SCTP)
```go
// Manual total, auto SCTP calculation
InitScenarioLogger(cfg, 20000, 0, httpSrv, ctx)
```
- `maxPool=20000`: Total capacity
- `nSctpWorker=0`: Auto-calculate SCTP as 1/6 of total (~3333)

## YAML Configuration Support (Optional Enhancement)

To add YAML configuration support, update `pkg/config/config.go`:

```go
type Config struct {
	GNodeBConfig GNodeBConfig `yaml:"gnodeb"`
	DefaultUe    UeConfig     `yaml:"defaultUe"`
	AMFs         []model.AMF  `yaml:"amfif"`
	Scenarios    []Scenario   `yaml:"scenarios"`
	RemoteServer RemoteServer `yaml:"remote"`
	Testing      TestingConf  `yaml:"testconf"`
	LogLevel     string       `yaml:"loglevel"`
	WorkerPool   WorkerPoolConfig `yaml:"workerpool,omitempty"`  // NEW
}

type WorkerPoolConfig struct {
	MaxPool       int  `yaml:"maxPool,omitempty"`       // 0 = auto-detect
	MaxSctpWorker int  `yaml:"maxSctpWorker,omitempty"` // 0 = auto-calculate
	AutoDetect    bool `yaml:"autoDetect,omitempty"`    // Force auto-detection
}
```

Then in `config.yml`:
```yaml
# Optional: Override worker pool configuration
# workerpool:
#   maxPool: 15000         # Total workers (0 = auto-detect based on CPU)
#   maxSctpWorker: 2500    # SCTP workers (0 = auto-calculate as maxPool/6)
#   autoDetect: true       # Force auto-detection even if values provided
```

## Hardware Sizing Guidelines

| Hardware | Auto-Detected Pool Size | Max UEs Recommended |
|----------|-------------------------|---------------------|
| 2 cores, 4GB RAM | ~3,000 workers | 500 UEs |
| 4 cores, 8GB RAM | ~6,000 workers | 1,500 UEs |
| 8 cores, 16GB RAM | ~12,000 workers | 3,000 UEs |
| 16 cores, 32GB RAM | ~24,000 workers | 6,000 UEs |
| 32 cores, 64GB RAM | ~48,000 workers | 12,000 UEs |

## Monitoring Pool Usage

Use `pool.GetPoolStats()` to monitor worker utilization:

```go
stats := pool.GetPoolStats()
fmt.Printf("MM Workers: %d/%d running\n", stats["mm_running"], stats["mm_capacity"])
fmt.Printf("SM Workers: %d/%d running\n", stats["sm_running"], stats["sm_capacity"])
fmt.Printf("SCTP Workers: %d/%d running\n", stats["sctp_running"], stats["sctp_capacity"])
```

Add to OAM backend (`monitoring/oambackend/`) for real-time monitoring via REST API.

## Troubleshooting

### Problem: "Too many workers, out of memory"
**Solution**: Reduce UE count or use manual pool sizing:
```go
InitScenarioLogger(cfg, 5000, 800, httpSrv, ctx)
```

### Problem: "Workers not running, pool saturated"
**Solution**: Check if SCTP workers are consuming entire pool. Auto-detection prevents this, but manual configs should follow: `maxPool/6 > nSctpWorker`

### Problem: "Performance degradation on high-core systems"
**Solution**: Auto-detection caps at 50,000 workers. For systems with >32 cores, manually increase:
```go
InitScenarioLogger(cfg, 80000, 13000, httpSrv, ctx)
```

## Migration Guide

### Before (Old Code)
```go
pool.InitWorkerPool(ctx, 20000, 20000)  // BUG: Zero MM/SM/GNB workers!
```

### After (New Code - Auto)
```go
pool.InitWorkerPool(ctx, 0, 0, nUEs, nGnbs)  // Auto-detects optimal config
```

### After (New Code - Manual)
```go
pool.InitWorkerPool(ctx, 15000, 2500, nUEs, nGnbs)  // Validates and adjusts if needed
```

## Algorithm Details

### Auto-Detection Formula
```
totalPool = min(max(numCPU * 1500, max(nUEs * 3 + nGnbs * 100, 1000)), 50000)
sctpWorkers = min(max(totalPool / 6, max(nGnbs * 50, 200)), totalPool / 3)
mmWorkers = (totalPool - sctpWorkers) * 40 / 100
smWorkers = (totalPool - sctpWorkers) * 35 / 100
gnbWorkers = (totalPool - sctpWorkers) - mmWorkers - smWorkers
```

This ensures:
1. Minimum viable pool (1,000 workers)
2. Scales with CPU cores
3. Scales with workload (UEs + gNodeBs)
4. Caps at reasonable maximum (50,000)
5. SCTP never consumes more than 1/3 of total pool
6. MM/SM/GNB always have workers available
