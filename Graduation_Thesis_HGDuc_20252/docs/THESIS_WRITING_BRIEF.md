# Thesis Writing Brief

Last reviewed: 2026-06-27

This file is the working brief for writing and revising the thesis end to end.
It summarizes the current thesis argument, the verified technical basis, and
the safest writing order.

## Core Thesis Argument

The thesis should be framed around a two-level mobility study:

1. A Python radio simulator generates structured 5G radio-measurement traces
   for terrestrial and aerial UE movement.
2. StormSIM replays measurement-driven serving-cell changes in a 5G protocol
   test environment and records handover delay, timer behavior, and failure
   outcomes.

The main technical distinction is:

- A radio-level serving-cell change is not a completed handover.
- A completed handover requires protocol execution involving UE, source gNB,
  target gNB, and the 5G Core.

This distinction should remain visible in every chapter, especially Chapter 4.

## System Under Study

### Radio Simulator

Main files:

- `main.py`
- `config.py`
- `hexagons.py`
- `mobility.py`
- `radio_models.py`
- `simulation.py`
- `logger.py`

Research role:

- Generate UE mobility and radio measurements.
- Use a hexagonal gNB layout.
- Use Manhattan mobility for implemented scenarios.
- Use UMa and UMa-AV propagation formulas through the UE-height regime.
- Log LOS probability, LOS/NLOS state, path loss, shadow fading, received
  power, RSRP, SINR, serving cell, neighbor RSRP, speed, and handover flag.

Important limitations:

- CLI does not expose seed.
- LOS/NLOS state is cached per UE-gNB link.
- Shadow fading is re-sampled after 25 m accumulated movement.
- RSRP is a system-level approximation from received power over 50 RBs and
  12 subcarriers per RB.
- Handover is an immediate RSRP-margin serving-cell change, with no
  time-to-trigger and no protocol delay.

### StormSIM / Code_HO

Main roles:

- Emulate UE and gNB behavior.
- Connect to 5G Core using N2/SCTP/NGAP and N3/GTP-U paths.
- Register UE and establish a PDU session before handover experiments.
- Replay measurement CSV rows.
- Trigger normal handover from `connected_gnb` changes.
- Execute Xn or N2 handover depending on the trace `Type` field.
- Export trial summaries, timer events, timer summaries, and phase logs.

Important limitations:

- Python output schema and StormSIM measurement input schema are not yet
  unified.
- Normal measurement mode does not select the target from RSRP; it replays the
  `connected_gnb` transition.
- The `pdr` telemetry field stores packet-loss probability, not packet
  delivery ratio.
- Current-source T304 is 1000 ms, while some test-plan notes mention 100 ms.
- Current-source Xn behavior does not fully match archived successful logs.
- CHO NTN position and coverage checks are placeholders.

## Current Chapter Status

### Front Matter

The abstract is aligned with the current thesis framing: radio-data generation
plus log-based handover-delay analysis. The abbreviations list already covers
the main terms.

Needs confirmation:

- Official English title.
- Whether the title should emphasize A2G, handover delay, StormSIM, or the
  complete measurement-driven workflow.

### Chapter 1

Status: strong.

Current function:

- Establishes the research context.
- States the key problem: radio measurements alone do not prove protocol-level
  handover completion.
- Defines objectives, scope, and contributions.

Writing stance:

- Keep Chapter 1 concise.
- Do not overclaim real-world validation.
- Keep the radio/protocol separation as the central motivation.

### Chapter 2

Status: mostly strong.

Current function:

- Explains cellular network concepts, 5G architecture, A2G communication,
  propagation, RSRP, SINR, handover delay, and open-source 5G cores.

Potential improvements:

- Make the mobility-model placement consistent with Chapter 3.
- Avoid broad 6G discussion unless it directly motivates A2G or dense mobility.
- Ensure every 3GPP formula claim has a citation.

### Chapter 3

Status: technically rich, but still has draft residue.

Current function:

- Describes the measurement-driven architecture.
- Defines the radio simulator design.
- Explains hexagonal layout, radio parameters, propagation flow, RSRP-based
  serving-cell decision, Manhattan mobility, and StormSIM handover execution.

Must fix before submission:

- Remove the red draft note in the StormSIM handover section.
- Verify claims against the current code before making final wording stronger.
- Clarify that the trace-normalization step is required because schemas differ.

### Chapter 4

Status: highest risk.

Current function:

- Defines experimental scenario groups.
- Gives altitude ranges H0-H3.
- Describes planned radio plots and handover metrics.
- Reports only two verified archived Xn runs.

Must fix before submission:

- Remove the red draft note.
- Replace "should include" / "should be evaluated" language with either
  actual results or explicitly scoped planned analysis.
- Confirm whether archived runs are official thesis results or debug logs.
- Do not present the two archived Xn runs as a complete loss-response curve.
- Keep warning that successful delay near 200 ms is affected by 200 ms polling.

### Chapter 5

Status: acceptable as a draft conclusion.

Current function:

- Summarizes workflow and selected findings.
- States limitations.
- Identifies future work.

Must finalize after Chapter 4:

- If new official experiments are added, update verified findings.
- If archived runs remain the only protocol result, keep limitations explicit.

### Appendices

Status: useful and aligned with the code.

Current function:

- Maps modules to research roles.
- Defines CSV fields and timer/trial output fields.
- Lists reproducibility requirements.

Potential improvements:

- Check final paths and schema names.
- Keep appendix factual rather than argumentative.

## Critical Confirmations Needed

Before finalizing Chapter 4 and Chapter 5, confirm:

1. Official thesis title in English.
2. Official radio datasets for H0, H1, H2, and H3.
3. Whether the generated data under `data/H*_...` is official or exploratory.
4. Exact command, seed, and config for each official radio run.
5. Whether Python radio output was converted into StormSIM measurement input,
   and if yes, which script or mapping was used.
6. Official T304 value: 100 ms or 1000 ms.
7. Whether archived runs `1781614531` and `1781622441` are thesis results.
8. Source revision and command used for the archived StormSIM runs.
9. Whether CHO is a main result or only future work.
10. Whether free5GC is the only 5G Core used in official runs.

## Recommended Writing Order

1. Clean obvious draft residue in Chapter 3 and Chapter 4.
2. Confirm the official title and result datasets.
3. Finalize the radio-result subsection in Chapter 4 using the H0-H3 data.
4. Finalize protocol-result subsection using only verified StormSIM runs.
5. Update Chapter 5 to match the final Chapter 4 evidence.
6. Rebuild the PDF and fix citations, captions, and list entries.
7. Read the whole PDF once for consistency of terminology:
   `StormSIM` vs `StormSim`, `gNB` vs `BS`, `loss probability` vs `PDR`.

## Safe Claim Boundaries

Safe claims:

- The project builds a configurable workflow for generating radio traces.
- The radio simulator supports A2G height regimes through UMa/UMa-AV formulas.
- The workflow separates radio-level serving-cell change from protocol-level
  handover completion.
- The selected archived Xn runs show lower success at higher configured loss.
- Delay values around 200 ms are affected by polling granularity.

Unsafe claims unless additional evidence is provided:

- Full validation against a commercial 5G deployment.
- Complete evaluation of NTN mobility.
- Complete loss-delay matrix.
- Complete Xn versus N2 comparison.
- Complete CHO performance evaluation.
- Exact protocol latency below the 200 ms polling interval.
- Reproducibility of archived logs from the current source revision.

