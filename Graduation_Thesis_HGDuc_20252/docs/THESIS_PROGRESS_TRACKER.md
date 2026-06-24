# Thesis Progress Tracker

Last updated: 2026-06-22

Use this file as the single progress table for thesis writing. Keep notes short so the table remains readable in VSCode.

Done marks:

- `[x]`: reviewed or finished for now.
- `[ ]`: still needs work.

## Progress Table

| Chapter | Section | Done | Status / Note |
|---|---|---:|---|
| Front matter | Title/Cover | [x] | Balanced into 3-line title on cover. |
| Front matter | Abstract | [x] | Updated to match simulation/log-based title. |
| Front matter | Foreword | [x] | Updated to match simulation/log-based title. |
| Chapter 1 | Research Context | [x] | Reviewed; citations added. Rebuild BibTeX. |
| Chapter 1 | Problem Statement | [x] | Reviewed; clarified radio vs protocol event gap. |
| Chapter 1 | Objectives | [x] | Reviewed; repeatability clarified for random mobility. |
| Chapter 1 | Scope | [x] | Reviewed; removed unclear controlled dataset wording. |
| Chapter 1 | Main Contributions | [x] | Reviewed; rewritten as concrete artifacts/results. |
| Chapter 1 | Chapter Conclusion | [x] | Reviewed; summarizes Chapter 1 and lists remaining chapters. |
| Chapter 2 | 5G/A2G Background | [x] | Reviewed; removed unsupported elevation-angle wording. |
| Chapter 2 | Mobility Model | [ ] | Draft; check flow and figures. |
| Chapter 2 | Propagation Model | [ ] | Draft; citations mostly present. |
| Chapter 2 | RSRP/SINR Theory | [ ] | Draft; cites TS 38.215. |
| Chapter 3 | Overall Design | [ ] | Draft; verify against code. |
| Chapter 3 | Radio Simulator | [ ] | Draft; check implementation claims. |
| Chapter 3 | StormSim/free5GC Workflow | [ ] | Draft; verify logs and configs. |
| Chapter 3 | Output Metrics | [ ] | Draft; check CSV/log field names. |
| Chapter 4 | Radio Results | [ ] | Needs revision; verify claims. |
| Chapter 4 | Handover Delay Results | [ ] | Needs revision; high overclaim risk. |
| Chapter 4 | Limitations | [ ] | Needs careful wording. |
| Chapter 5 | Conclusion | [ ] | Draft; finalize after Chapter 4. |
| All chapters | Chapter Conclusion convention | [x] | Chapter 2-5 endings standardized. |
| Appendices | File Inventory | [ ] | Draft; check final paths. |
| Appendices | CSV Fields | [ ] | Draft; check final schemas. |
| References | Bibliography | [ ] | Added TS 38.300; rebuild needed. |
| Build | PDF/BibTeX | [ ] | Build tools unavailable in current shell. |

## Citation Notes

| Claim | Citation | Status / Note |
|---|---|---|
| RSRP/SINR are NR physical-layer measurements. | `3gpp_38215` | Verified from TS 38.215. |
| Handover involves source gNB, target gNB, UE, and 5GC-side procedure. | `3gpp_38300` | Verified from TS 38.300 clause 9.2.3.2 and clause 4.3. |
| Propagation concepts include path loss, LOS probability, and shadow fading. | `3gpp_38901`, `3gpp_36777` | Used in Chapter 2/3. |
| 5G service categories. | `itu_m2083` | Used in Chapter 2. |
| Real-world LTE channel-quality traces exist and can include RSRP/RSRQ. | `meixner2018lte_trace` | Used in Problem Statement. |

## Open Issues

| Issue | Status / Note |
|---|---|
| Citation numbering | Rebuild after removing `\nocite{*}` and fixing captions. |
| Captions with citations | Use optional short captions without `\cite{}`. |
| Chapter 4 reproducibility | Review carefully before final claims. |
| Unrelated pycache changes | Existing modified files; not part of thesis writing. |

## Update Log

| Date | Section | Change | Status |
|---|---|---|---|
| 2026-06-22 | Chapter 1 - Research Context | Added `3gpp_38215` and `3gpp_38300`; fixed citation-order issue. | Reviewed |
| 2026-06-22 | Chapter 1 - Problem Statement | Reframed dataset limit and clarified radio measurements vs protocol handover events. | Reviewed |
| 2026-06-22 | Chapter 1 - Problem Statement | Removed repeated handover-delay paragraph and kept radio/protocol separation as the core problem. | Reviewed |
| 2026-06-22 | Chapter 1 - Objectives | Replaced absolute reproducibility with experiment-design-level repeatability. | Reviewed |
| 2026-06-22 | Chapter 1 - Scope | Replaced controlled dataset phrasing with structured data from explicit simulation settings. | Reviewed |
| 2026-06-22 | Chapter 1 - Main Contributions | Rewritten to avoid repeating Objectives and to focus on concrete contributions. | Reviewed |
| 2026-06-22 | Chapter endings | Added/updated chapter conclusions; Chapter 1 conclusion includes thesis organization. | Reviewed |
| 2026-06-22 | Front matter | Added title to cover/declaration and aligned Abstract/Preface wording. | Reviewed |
| 2026-06-22 | Front matter | Balanced cover title and changed connector from with to and. | Reviewed |
| 2026-06-22 | Chapter 2 - A2G Background | Removed elevation-angle claim and cited altitude/LOS/interference wording. | Reviewed |
