# Thesis Progress Tracker

Last updated: 2026-06-22

Use this file as the single progress table for thesis writing. Keep notes short so the table remains readable in VSCode.

Done marks:

- `[x]`: reviewed or finished for now.
- `[ ]`: still needs work.

## Progress Table

| Chapter | Section | Done | Status / Note |
|---|---|---:|---|
| Front matter | Abstract | [ ] | Draft; final pass after Chapter 4. |
| Front matter | Foreword | [ ] | Draft; check overlap with Introduction. |
| Chapter 1 | Research Context | [x] | Reviewed; citations added. Rebuild BibTeX. |
| Chapter 1 | Problem Statement | [ ] | Draft; logic is acceptable. |
| Chapter 1 | Objectives | [ ] | Draft; check against Chapter 4/5. |
| Chapter 1 | Scope | [ ] | Draft; verify no overclaiming later. |
| Chapter 1 | Main Contributions | [ ] | Draft; compare with actual results. |
| Chapter 1 | Thesis Organization | [ ] | Draft; simple structure statement. |
| Chapter 2 | 5G/A2G Background | [ ] | Draft; citation pass needed. |
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
