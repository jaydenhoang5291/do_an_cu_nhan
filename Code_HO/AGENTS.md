# StormSIM Project Knowledge Base

## Project Overview
5G UE and gNodeB emulator for testing and benchmarking 5G Core networks (Open5GS, Free5GC). Designed for extreme scale (10,000+ UEs, 100+ gNBs) using custom worker pools and virtual transport.

## Key Technologies
- **Language**: Go 1.22.5+
- **Networking**: SCTP (N2 Interface), Netlink for GTP management
- **User Plane**: GTP5G (Kernel-level GTP-U management)
- **Observability**: High-performance Ring Buffers for concurrent UE/gNB logging, delay tracking

## System Architecture
```
                    ┌─────────────────────────────────────────────────────┐
                    │                    AMF (5G Core)                    │
                    └───────────────────────┬─────────────────────────────┘
                                            │ SCTP/NGAP (N2)
                    ┌───────────────────────▼─────────────────────────────┐
                    │                  GnbContext                         │
                    │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐ │
                    │  │   AMF Pool   │  │   UE Pool    │  │ NGAP Disp. │ │
                    │  │  (sync.Map)  │  │  (sync.Map)  │  │ (handlers) │ │
                    │  └──────────────┘  └──────────────┘  └────────────┘ │
                    └───────────────────────┬─────────────────────────────┘
                                            │ RLink (Virtual Radio)
                    ┌───────────────────────▼─────────────────────────────┐
                    │               UeContext (per UE)                    │
                    │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐ │
                    │  │  5GMM FSM    │  │  5GSM FSM    │  │ NAS/Timer  │ │
                    │  │ (MmWorkerPool│  │ (SmWorkerPool│  │  Engine    │ │
                    │  └──────────────┘  └──────────────┘  └────────────┘ │
                    └─────────────────────────────────────────────────────┘
```

## Critical Constraints
- **Concurrency**: NO mutexes in hot paths. Use `sync/atomic` for ID generation and counters. Use channels for coordination.
- **Memory Safety**: NO `sync.Pool` for SCTP message reads to avoid data corruption across worker goroutines.
- **FSM Flow**: NO recursive FSM events. Never call `SendEvent` from within an FSM callback; use `SetNextEvent` instead.

## Anti-patterns (Technical Debt)
- **Race Conditions**: Known race conditions in ID generation (`getRanUeId`, `getUeTeid`) under saturated load.
- **Error Handling**: Excessive `Fatal()` calls in protocol handlers instead of graceful recovery.

## Core Commands
```bash
# Build Emulator & Client
make

# Manual gogtp5g Build
go build -o bin/gogtp5g-link ./monitoring/gtp5g/gogtp5g-link
go build -o bin/gogtp5g-tunnel ./monitoring/gtp5g/gogtp5g-tunnel

# Run Tests
go test ./...
```

## Repository Map
| Path | Purpose |
|------|---------|
| `internal/core/uecontext/` | UE context, 5GMM/5GSM FSM, NAS handlers |
| `internal/core/gnbcontext/` | gNB context, NGAP handlers, AMF pool |
| `internal/common/fsm/` | Async FSM framework |
| `internal/common/pool/` | Worker pool management (Mm/Sm/Gnb/Sctp) |
| `internal/common/logger/` | Ring buffer logging, delay tracking |
| `internal/common/stats/` | Procedure statistics |
| `internal/transport/rlink/` | Virtual radio (UE↔gNB channels) |
| `internal/transport/sctpngap/` | SCTP/NGAP transport |
| `internal/scenarios/` | UE grouping, event distribution |
| `pkg/model/` | Shared 3GPP data models, events, states |
| `monitoring/oambackend/` | REST API for control/metrics |
| `monitoring/gtp5g/` | Kernel GTP-U tunnel management |

## Subdirectories with AGENTS.md

### Commands
- `./cmd/AGENTS.md` - Command entry points overview
- `./cmd/client/AGENTS.md` - OAM CLI client
- `./cmd/emulator/AGENTS.md` - Emulator main entry point

### Configuration
- `./config/AGENTS.md` - YAML configuration files
- `./pkg/config/AGENTS.md` - Configuration parsing

### Shared Libraries
- `./pkg/AGENTS.md` - Public packages overview
- `./pkg/model/AGENTS.md` - 3GPP data models

### Internal Core
- `./internal/AGENTS.md` - Internal packages overview
- `./internal/core/AGENTS.md` - Core context overview
- `./internal/core/gnbcontext/AGENTS.md` - gNB context, NGAP handlers
- `./internal/core/uecontext/AGENTS.md` - UE context, NAS handlers
- `./internal/core/uecontext/sec/AGENTS.md` - 5G AKA security
- `./internal/core/uecontext/timer/AGENTS.md` - NAS timers
- `./internal/core/managers/AGENTS.md` - Handover management, NTN coordination

### Internal Common
- `./internal/common/AGENTS.md` - Shared infrastructure overview
- `./internal/common/ds/AGENTS.md` - Data structures
- `./internal/common/fsm/AGENTS.md` - Async FSM framework
- `./internal/common/logger/AGENTS.md` - Ring buffer logging, delay tracking
- `./internal/common/pool/AGENTS.md` - Worker pools
- `./internal/common/stats/AGENTS.md` - Procedure statistics

### Transport
- `./internal/transport/AGENTS.md` - Transport overview
- `./internal/transport/rlink/AGENTS.md` - Virtual radio (UE↔gNB)
- `./internal/transport/sctpngap/AGENTS.md` - SCTP/NGAP transport

### Scenarios
- `./internal/scenarios/AGENTS.md` - UE grouping, event distribution

### Monitoring
- `./monitoring/AGENTS.md` - Observability overview
- `./monitoring/gtp5g/AGENTS.md` - GTP tunnel management
- `./monitoring/gtp5g/gogtp5g-link/AGENTS.md` - GTP link tool
- `./monitoring/gtp5g/gogtp5g-tunnel/AGENTS.md` - GTP tunnel tool
- `./monitoring/oambackend/AGENTS.md` - REST API / CLI handlers
- `./monitoring/resource/AGENTS.md` - Resource monitoring

### Documentation
- `./docs/AGENTS.md` - Project documentation

