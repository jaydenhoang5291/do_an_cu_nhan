# Thesis Progress Tracker

Last updated: 2026-06-22

Purpose: keep one lightweight record of thesis progress so section status, citations, and open issues do not need to be rediscovered.

Status tags:

- `[Draft]` content exists, still needs review.
- `[Reviewed]` logic and thesis fit have been checked.
- `[Needs citation]` citation should be added or verified.
- `[Needs revision]` known content issue exists.
- `[Ready]` acceptable except final formatting/build checks.

## Chapter 1 - Introduction

- [Reviewed] Research Context
  - File: `body/chap1-introduction.tex`
  - Current note: logic is good.
  - Citations added:
    - `3gpp_38215` for NR physical-layer measurements such as RSRP/SINR.
    - `3gpp_38300` for protocol-level handover involving source gNB, target gNB, UE, and 5GC.
  - Next: rebuild bibliography and verify numbering.

- [Draft] Problem Statement
  - File: `body/chap1-introduction.tex`
  - Current note: separates dataset availability, logging limits, and radio/protocol distinction.
  - Next: review wording after Research Context is final.

- [Draft] Objectives
  - File: `body/chap1-introduction.tex`
  - Current note: aligned with simulation plus handover-delay workflow.
  - Next: check that Chapter 4 and Chapter 5 answer each objective.

- [Draft] Scope
  - File: `body/chap1-introduction.tex`
  - Current note: limits real-world campaign and commercial optimization claims.
  - Next: verify no later chapter overclaims beyond this scope.

- [Draft] Main Contributions
  - File: `body/chap1-introduction.tex`
  - Current note: clear but concise.
  - Next: compare with actual implemented results.

## Chapter 2 - Theoretical Background

- [Draft] Overall chapter
  - File: `body/chap2-theoretical_background.tex`
  - Current note: covers 5G, A2G, mobility, propagation, RSRP, and SINR.
  - Citation-order note: table caption now has an optional short caption without citation.
  - Next: check citation order after rebuilding BibTeX.

## Chapter 3 - System Design

- [Draft] Overall chapter
  - File: `body/chap3-system_design.tex`
  - Current note: describes Python simulator and StormSim/free5GC workflow.
  - Citation-order note: table caption already has an optional short caption without citation.
  - Next: verify implementation claims against current code and logs.

## Chapter 4 - Experimental Results

- [Needs revision] Overall chapter
  - File: `body/chap4-experimental_results.tex`
  - Current note: mentions archived logs may not match current source revision.
  - Risk: this chapter has the highest overclaiming risk.
  - Next: decide final result scope and phrase limitations clearly.

## Chapter 5 - Conclusion

- [Draft] Overall chapter
  - File: `body/chap5-conclusion.tex`
  - Current note: should reflect Chapter 4 limitations.
  - Next: finalize only after Chapter 4 is stable.

## Front Matter And Appendices

- [Draft] Abstract
  - File: `Abstract.tex`
  - Current note: consistent with current thesis direction.
  - Next: final pass after Chapter 4 is stable.

- [Draft] Foreword
  - File: `LoiNoiDau.tex`
  - Current note: matches high-level motivation.
  - Next: check overlap with Introduction.

- [Draft] Appendices
  - File: `body/appendices.tex`
  - Current note: documents file inventory and CSV fields.
  - Next: check consistency with final source files.

## References And Build

- [Needs citation] References
  - File: `references.bib`
  - Current note: `3gpp_38300` was added.
  - Next: rebuild BibTeX and inspect final numbering.

- [Needs revision] Build output
  - Current note: build tools were not available in the current shell.
  - Next: run a clean LaTeX/BibTeX build in an environment with `pdflatex` or `latexmk`.

## Citation Checks

- Claim: RSRP/SINR are NR physical-layer measurements.
  - Citation: `3gpp_38215`
  - Location: `body/chap1-introduction.tex`
  - Verified against: 3GPP TS 38.215, SS-RSRP, CSI-RSRP, SS-SINR, CSI-SINR definitions.

- Claim: handover is a protocol procedure involving source gNB, target gNB, UE, and 5GC-side signaling/path switch.
  - Citation: `3gpp_38300`
  - Location: `body/chap1-introduction.tex`
  - Verified against: 3GPP TS 38.300 clause 9.2.3.2 for handover steps and clause 4.3 for NG-RAN/Xn/5GC connectivity.

- Claim: propagation concepts include path loss, LOS probability, and shadow fading.
  - Citations: `3gpp_38901`, `3gpp_36777`
  - Location: Chapter 2 and Chapter 3.
  - Next: keep as-is unless formulas are changed.

## Open Issues

- Rebuild bibliography after citation-order fixes.
- Keep citations out of list captions by using optional short captions.
- Review Chapter 4 carefully before claiming reproducible final results.
- Existing `__pycache__` modified files are unrelated to thesis writing changes.

## Update Log

Use this short format when a section changes:

```text
Date:
Section:
Change:
Status after change:
Open issue:
```
