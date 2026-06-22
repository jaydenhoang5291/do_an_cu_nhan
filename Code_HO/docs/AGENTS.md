# Documentation Directory

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

## Purpose

This directory contains comprehensive documentation for the StormSIM 5G UE/gNB emulator project. Documentation covers system architecture, configuration, features, testing methodologies, research context, and technical implementation guides.

## Key Files

| File | Description |
|------|-------------|
| `ARCH.md` | System architecture - component overview, data flow, state machines (5GMM/5GSM), concurrency model, module reference, transport layer, and technical debt |
| `CONFIGURATION.md` | Configuration guide - gNodeB/UE setup, AMF interfaces, test scenarios, remote API, testing options, and troubleshooting |
| `FEATURES.md` | Feature documentation - 5G procedures (registration, PDU sessions, handover), security algorithms, fuzzy testing, chaos engineering, scalability |
| `RESEARCH.md` | Academic context - research objectives, methodology, contributions, timeline for the 5G core benchmarking project |
| `TESTING.md` | Testing guide - conformance tests, performance/load tests, handover scenarios, reliability tests, execution commands, metrics collection |
| `WORKER_POOL_GUIDE.md` | Worker pool configuration - auto-detection algorithm, hardware sizing, subpool allocation (MM/SM/GNB/SCTP), troubleshooting |

## Documentation Structure

```
docs/
├── ARCH.md              # Architecture deep-dive
├── CONFIGURATION.md     # YAML configuration reference
├── FEATURES.md          # Capabilities and 5G procedures
├── RESEARCH.md          # Academic/research background
├── TESTING.md           # Test scenarios and execution
├── WORKER_POOL_GUIDE.md # Worker pool technical guide
├── arch.png             # Architecture diagram image
├── expr.png             # Experiment setup diagram
├── pc.drawio            # Architecture diagram (draw.io source)
├── ideas/               # Research brainstorming (fuzz testing, RL, conformance)
├── idea2/               # Research iteration 2 (5GC-Fuzz, AMFuzz, CovFUZZ, 5GReplay)
└── idea3/               # Research iteration 3 (comprehensive guide)
```

## For AI Agents

### When to Update These Files

- **ARCH.md**: When adding new components, changing state machines, modifying concurrency patterns, or discovering new technical debt
- **CONFIGURATION.md**: When adding new YAML configuration options or changing default values
- **FEATURES.md**: When implementing new 5G procedures, security algorithms, or testing capabilities
- **RESEARCH.md**: Generally static - update only for major research direction changes
- **TESTING.md**: When adding new test scenarios, changing test commands, or updating success criteria
- **WORKER_POOL_GUIDE.md**: When modifying pool allocation algorithm or hardware sizing recommendations

### Documentation Style Guidelines

1. **Architecture diagrams**: Use ASCII art for diagrams (see `ARCH.md` for examples)
2. **Code examples**: Use fenced code blocks with language tags (`yaml`, `go`, `bash`)
3. **Tables**: Use markdown tables for comparisons and parameter references
4. **Cross-references**: Link to other docs using relative paths (e.g., `[CONFIGURATION.md](./CONFIGURATION.md)`)

### Key Cross-References

- Architecture → Configuration: Component settings map to YAML sections
- Features → Testing: Each feature should have corresponding test scenarios
- Configuration → Testing: YAML examples should match test configurations

## Dependencies

**None** - Documentation is standalone. However, the content references:
- `config/config.yml` - Configuration file examples
- `internal/` directories - Implementation code paths
- 3GPP specifications (TS 24.501, TS 38.413) - External standards

---

*Documentation index for StormSIM 5G emulator*
