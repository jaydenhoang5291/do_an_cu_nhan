# Technical inventory of `Code` and `Code_HO`

This document complements `PROJECT_RESEARCH_NOTES.md`. It maps source modules to research functions, inputs, state, outputs, and known limitations. Dependency code under `Code_HO/venv` is excluded because it is not project-authored research code.

## 1. `Code`: Python radio simulator

### `main.py`

- Entry point for interactive simulation.
- Reads rectangle length/width, road spacing, UE count, optional aerial height, and link-line visibility.
- Creates `CellularNetworkReceivedPower` with `fast_mode=True`.
- Does not expose the constructor's random seed, simulation duration, or radio constants through the CLI.

### `config.py`

Central source of radio constants:

- link budget: `PTX`, `GTX`, `GRX`;
- carrier and receiver: `FC`, `BANDWIDTH`, `UE_NOISE_FIGURE`;
- association: `HOM`;
- RSRP approximation: `LTE_N_RB`, `LTE_N_SUBCARRIERS_PER_RB`;
- heights and supported UE range;
- ground-UE shadow-fading standard deviations;
- shadow-fading update distance;
- stop duration, turn threshold, and acceleration duration;
- simulation steps and time per step.

Changing this file changes all runs unless a class attribute is overridden after construction.

### `hexagons.py`

- Defines cube-coordinate `Hex(q,r,s)` with invariant `q+r+s=0`.
- Converts axial coordinates to Cartesian BS positions.
- Generates enough hexagons to cover the rectangular study area plus a half-height buffer.
- Returns topology indices only; `simulation.py` adds the canvas-center offset and creates gNB state.

### `mobility.py`

- Creates sorted road-coordinate arrays, explicitly including both region boundaries.
- Determines valid cardinal directions at an intersection.
- Selects a new direction uniformly from all valid directions, including continuation and reversal.
- Increments the turn count only when the selected direction differs from the previous direction.
- Stops a UE after the configured number of direction changes.
- Consumes a movement distance across multiple road segments in one simulation step.
- Clips final position to the rectangular region.

Research consequence: no probability preference is assigned to straight movement, left/right turns, or U-turns.

### `ue.py`

- Defines `AerialUE` metadata: height, speed, direction, and type.
- Validates height against 1.5--300 m.
- The simulator's authoritative movement state remains in arrays owned by `CellularNetworkReceivedPower`; this dataclass is metadata rather than an independent mobility actor.

### `radio_models.py`

Pure/model functions:

- UMa and UMa-AV LOS probability;
- effective breakpoint distance;
- minimum effective 2D distance and 3D distance;
- UMa LOS/NLOS path loss;
- UMa-AV LOS/NLOS path loss;
- height-dependent aerial shadow-fading sigma.

`RadioModel` responsibilities:

- calculate link distances using per-UE height and per-BS height;
- create and retain per-link LOS state;
- create and update per-link shadow fading;
- calculate total received power and RSRP;
- calculate receiver thermal noise;
- select six first-tier interferers around the serving gNB;
- calculate SINR;
- rank all gNBs and apply the 3 dB serving-cell margin.

Important state behavior:

- LOS state is retained in `sf_cache` for the full link lifetime.
- Shadow fading is retained until accumulated movement reaches 25 m.
- Candidate association considers every generated gNB, not only the six logged neighbors.
- The six logged neighbors are the six strongest non-serving RSRP candidates.
- The six SINR interferers are selected by distance to the serving gNB, not by strongest interference at the UE.

### `logger.py`

- Creates a column-oriented in-memory log.
- Adds one row per simulation frame and one field group per UE.
- Stores six neighbor index/RSRP pairs.
- Pads unequal column lengths before DataFrame creation.
- Rounds radio and position quantities to two decimal places.
- Writes UTF-8 CSV with a timestamp and UE count in the filename.

### `simulation.py`

System orchestrator:

- validates input and initializes random state;
- builds the region, roads, radio model, logger, and topology;
- samples UE speeds and starting intersections;
- maintains serving-cell, handover, turn, pause, ramp, and fading state;
- draws the interactive Matplotlib view;
- updates all UEs once per frame;
- saves one combined CSV at successful completion.

The loop uses `range(start_frame, steps + 1)`, so `SIMULATION_STEPS=300` produces 301 rows.

### `utils.py`

Numeric input helpers. Invalid input silently falls back to defaults rather than terminating.

## 2. `Code_HO`: StormSIM protocol emulator

## 2.1. Entrypoints

### `cmd/emulator/emulator.go`

Selects execution mode:

- normal configured scenarios;
- replay mode;
- event-based CSV handover;
- measurement-based CSV handover;
- optional PCAP capture;
- optional fail mode;
- optional CHO mode.

It maps YAML simulation fields into CSV scenario structs. The command does not expose `T304DurationMs` or handover-command retry count even though those fields exist in `MeasurementCsvConfig`.

### `cmd/client/client.go`

Client-side entrypoint for interacting with the remote/OAM API. It is ancillary to the thesis handover data path.

## 2.2. Configuration and models

### `pkg/config`

- `config.go`: root YAML schema, scenario events, impairment fields, remote API, fuzz settings, and logging defaults.
- `gnbconf.go`: N2/N3 interfaces and list of gNBs; resolves hostnames to IPv4.
- `ueconf.go`: subscriber identity, authentication material, DNN, HPLMN, S-NSSAI, security algorithms, and inter-UE event delay.
- `csvreader.go`: parses both handover-event and measurement CSV formats.

Normal measurement parsing:

- detects gNB columns only when names start with `gnb` and end with `_rsrp`;
- calculates a handover trigger solely from a change in `connected_gnb`;
- defaults missing `Type` to Xn;
- retains RSRP values for CHO evaluation or diagnostics.

### `pkg/model`

Contains shared protocol and state data:

- UE security/ciphering and tunnel mode;
- gNB/AMF interface and PLMN models;
- S-NSSAI and PDU-session contexts;
- 5GMM, 5GSM, handover-monitor, and timer event/state identifiers;
- virtual-radio messages for NAS, paging, PDU setup, context transfer, and handover;
- CHO candidates, A3/A5/NTN conditions, measurement reports, and execution notifications.

## 2.3. Common execution infrastructure

### `internal/common/fsm`

Asynchronous finite-state-machine framework. Transitions are indexed by `(current state, event)` and callbacks execute through worker pools.

### `internal/common/pool`

Creates worker pools used by MM, SM, gNB, SCTP, and handover processing. This supports concurrent UE execution.

### `internal/common/ds`

Generic task queue infrastructure used to feed UE events.

### `internal/common/logger`

Provides:

- structured and buffered logs;
- ring buffers;
- NAS/NGAP procedure and delay types;
- timer event storage;
- CSV/NDJSON export;
- timer lifecycle pairing and percentile helper functions.

Timer-event structs define more fields than current scenarios populate. Empty `rsrp_at_trigger`, RLF, beam, and N2 RTT fields must not be interpreted as measured zeros.

### `internal/common/stats`

Stores procedure histories and aggregate statistics for OAM and diagnostic reporting.

## 2.4. Virtual and external transport

### `internal/transport/rlink`

`Connection` represents the virtual UE--gNB radio link:

- independent uplink/downlink channels;
- loss probability in each direction;
- fixed delay and symmetric bounded jitter;
- 3 s queue-send timeout by default;
- sent, delivered, dropped, and timeout counters.

Loss is sampled independently with `rand.Float64() < loss`. Jitter is sampled in approximately `[-jitter/2,+jitter/2)`, and negative total delay is clamped to zero.

`SendDownlinkCritical` and `SendUplinkCritical` bypass loss, jitter, and configured delay. Therefore, results depend on whether a message is sent through a normal or critical path.

### `internal/transport/sctpngap`

Maintains SCTP transport and NGAP identifiers for communication between each emulated gNB and AMF.

## 2.5. UE side

### `internal/core/uecontext`

Major responsibilities:

- creation and lifecycle of each UE;
- 5GMM registration/deregistration/service state machine;
- 5GSM PDU-session state machine;
- NAS message encoding/decoding;
- authentication, Milenage, key derivation, integrity, and ciphering state;
- virtual-radio message handling;
- paging and idle transitions;
- PDU-session and optional GTP setup;
- handover command handling;
- radio-link-failure and recovery timer triggers;
- measurement storage and CHO evaluation.

Handover handling:

1. Receive `RLinkHandoverPrepareRequest`.
2. Start T304.
3. Sleep for the handover connection's configured downlink delay to emulate random access/RRC execution.
4. Abort if T304 expired; otherwise respond, close the old link, switch gNB/link, and stop T304.
5. On T304 expiry, start T311; on T311 expiry, start T301.

CHO:

- candidate evaluation runs every 500 ms;
- default condition without explicit rules is A3;
- A5 is available;
- time-to-trigger fields are modeled but not enforced by the evaluator;
- NTN position and coverage checks currently return true.

## 2.6. gNB side

### `internal/core/gnbcontext`

Major responsibilities:

- create gNB contexts and connect to AMFs;
- own UE contexts and PDU-session state;
- encode, send, dispatch, and handle NGAP procedures;
- forward NAS between UE and AMF;
- execute Xn and N2 handover;
- send path-switch and context-release signaling;
- manage per-gNB radio/Xn impairment and timers;
- prepare CHO candidates.

Xn trigger behavior in the current source:

- creates the target virtual link;
- sets its `DownlinkDelay` to `T304 + 500 ms`;
- starts TXnRELOCprep (3 s) and TXnRELOCoverall (6 s);
- attempts context forwarding up to three times;
- waits 50 ms between failed attempts;
- logs `transport_fail` if all attempts fail;
- keeps the preparation timer running to permit a natural timeout;
- cancels the overall timer after transport failure;
- sends the handover command through a critical downlink path.

This current-source delay assignment is intentionally larger than T304 and is expected to exercise failure behavior. It is inconsistent with archived runs in which many T304 events stop successfully after tens of milliseconds.

TODO: cần xác nhận từ người dùng. Identify the exact source revision and command used to produce each official archived result.

N2 trigger behavior:

- builds and sends NGAP `HandoverRequired`;
- requires at least one PDU session;
- starts TNGRELOCprep and TNGRELOCoverall;
- target preparation and UE command are handled through NGAP request/command handlers;
- path-switch acknowledgement stops NG relocation timers.

## 2.7. Handover monitoring

### `internal/core/monitor`

Contains two related handover representations:

- a manager-level FSM with preparation, execution, completion, and null states;
- `HoProcedure`, used by the measurement scenario to timestamp prepare, execute, complete, success, and failure.

It also includes:

- handover queue and result tracker;
- gNB grouping and network-condition metadata;
- NTN coordinator and visibility windows;
- OAM endpoints for statistics, groups, satellites, visibility, and target queries.

The NTN coordinator stores configured visibility windows and does not calculate satellite trajectories itself.

## 2.8. Scenario orchestration

### `internal/scenarios/test-with-custom-scenarios.go`

Runs YAML-defined UE groups and events for registration, sessions, handover, replay/fuzz, and general emulator testing.

### `internal/scenarios/test-single-ue.go`

Initializes shared logging, creates gNBs, and runs a single-UE or replay scenario.

### `internal/scenarios/groupUE.go`

Creates groups of UEs, increments subscriber IDs, distributes configured events with delay, and balances initial gNB assignment.

### `internal/scenarios/test-csv-handover.go`

Reads explicit handover rows, creates one UE, completes registration/PDU setup at zero loss, restores impairment, triggers each Xn/N2 handover, and polls completion every 500 ms with a 15 s outer timeout.

### `internal/scenarios/test-measurement-ho.go`

Main thesis-related measurement scenario:

- reads every measurement row;
- selects the first `connected_gnb` as the initial cell;
- registers the UE and establishes one PDU session at zero loss;
- restores radio and Xn impairment;
- triggers when `connected_gnb` changes;
- supports normal or CHO execution;
- polls target success every 200 ms;
- uses an outer timeout of `max(5*T304, 2 s)`;
- hooks UE and gNB timer engines;
- exports raw timer events, paired timer summaries, trial summaries, RLink statistics, and phase timestamps.

The polling interval limits timing resolution. A reported value near 200 ms may mean completion occurred at any time before the first successful poll.

### `internal/scenarios/handover_tracker.go`

Stores one result per triggered handover: step, source, target, type, start/end time, duration, success, and reason.

### `internal/scenarios/ho_phase_csv.go`

Exports absolute timestamps for monitor phase entry. It does not directly export per-phase durations; those must be calculated from adjacent timestamps.

### `internal/scenarios/remote-api.go`

Exposes UE, gNB, session, worker, and delay information to the monitoring backend.

## 2.9. Monitoring

### `monitoring/oambackend`

REST/OAM models and handlers for emulator state, UE/gNB information, statistics, and watchers.

### `monitoring/pcap.go`

Optional packet capture integration.

### `monitoring/resource`

Python utilities/notebook for CPU and RAM observation. These are not used in the reported handover result calculation.

### `monitoring/gtp5g`

Bundled user-plane tunnel/link management utilities. They support GTP-U operation but are not the source of the archived handover summary metrics.

## 2.10. Data generation and analysis

### `data/data.py`

Generates a synthetic 100-step, five-gNB RSRP file with a target number of serving-cell transitions. It constrains per-step RSRP change and attempts to enforce a target margin. The sequence is synthetic and must not be described as measured field data.

### `data/goocs2_grid.py`

Earlier/alternative combined grid and UAV-BS simulator. Its propagation assumptions and CSV schema differ from the current modular `Code` simulator. It should not be used as evidence for current `Code` formulas without explicit selection.

### `scripts/analyze_timers.py`

Checks raw timer files for unmatched starts/stops and elapsed-time disagreement, and emits anomaly and paired-summary CSV files.

### `analyze_timer_events.py`

Specialized analysis of TXnRELOCprep stop, natural timeout, and transport-failure events.

### `logs/plot_comparison.py`

Plots procedure/NAS comparison logs for 5G Core implementations. It is ancillary to the selected handover runs.

## 3. End-to-end field mapping

| Research concept | `Code` output | StormSIM input/output |
|---|---|---|
| Time step | `Step` | `Step`/`Bước`, then `trial_id` |
| Serving cell | `ue0_connected_bs` | input `connected_gnb` |
| Neighbor RSRP | `ue0_bsN_idx`, `ue0_bsN_rsrp` | input requires fixed `gnbX_rsrp` columns |
| Radio HO event | `ue0_handover` | normal measurement mode recomputes from serving-cell change |
| HO type | not generated | optional input `Type`, default Xn |
| Link impairment | not generated | YAML loss, delay, jitter |
| Overall outcome | not modeled | `overall_result`, `fail_reason` |
| HO duration | not modeled | `total_ho_ms` |
| Timer lifecycle | not modeled | timer-event and timer-summary CSV |

## 4. Reproducibility requirements for final experiments

For an official result set, record together:

1. source-code revision or immutable archive;
2. exact command line;
3. YAML configuration;
4. input measurement CSV;
5. random seed where applicable;
6. T304 and all other timer values;
7. raw timer event file;
8. trial summary file;
9. schema version and interpretation of loss/PDR;
10. expected trial count and a lifecycle-completeness check.

