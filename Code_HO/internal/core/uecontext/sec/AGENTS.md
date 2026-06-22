<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# sec

## Purpose
NAS security package implementing 5G AKA authentication, key derivation functions (KDF), and the Milenage algorithm for UE context. Handles cryptographic operations including encryption/integrity keys for NAS messages, AS key derivation for handover (KGNB, NH), and sequence number management for authentication synchronization per 3GPP TS 33.501 and TS 33.220.

## Key Files
| File | Description |
|------|-------------|
| `secctx.go` | `SecurityContext` struct - NAS/AS key derivation, 3GPP/Non-3GPP bearer contexts, KGNB/KN3IWF/NH for handover |
| `ueauth.go` | 5G AKA KDF functions (KAMF, KSEAF, KAUSF, RES*, CK'/IK') with FC constants per TS 33.501 Annex A |
| `milenage.go` | Milenage algorithm (F1-F5*) using AES for 5G AKA authentication, OP/OPC support, AUTS validation |
| `sqn.go` | `Sqn` struct - Sequence number management for authentication sync and resynchronization |
| `milenage_test.go` | Test vectors for Milenage F1-F5* functions |
| `ueauth_test.go` | KDF test cases with RFC 5448 test vectors |

## For AI Agents

### Working In This Directory
- **Cryptographic Safety**: All key derivation uses HMAC-SHA256 via standard library. Do not modify KDF implementations.
- **Key Storage**: SecurityContext stores keys as byte slices. Never log or expose key material.
- **OP vs OPC**: Milenage can use either OP (Operator Variant) or pre-computed OPC. Use `isopc=true` when providing OPC directly.
- **Random Reader**: Milenage supports custom `io.Reader` for deterministic testing via `NewMilenageEx()`.

### Common Patterns

**Creating Security Context:**
```go
// After successful 5G AKA authentication
secCtx := sec.NewSecurityContext(&ngKsi, kamf, false) // false = UE side
err := secCtx.DeriveNasKeys(encAlg, intAlg, sec.HDP_NONE)
```

**Milenage Authentication:**
```go
m, err := sec.NewMilenage(k, op, false)  // false = OP, not OPC
m.SetRand(randFromAmf)
maca, macs, _ := m.F1(sqn, amf)  // MAC-A, MAC-S*
res, ak := m.F2F5()               // RES, AK
ck := m.F3()                      // CK
ik := m.F4()                      // IK
akstar := m.F5star()              // AK*
```

**Key Derivation Functions:**
```go
// KAUSF derivation (TS 33.501 Annex A.2)
kausf, err := sec.KAUSF(ckik, servingNetworkName, sqnXorAk)

// KSEAF derivation (TS 33.501 Annex A.4)
kseaf, err := sec.SeafKey(kausf, supi)

// KAMF derivation (TS 33.501 Annex A.5)
kamf, err := sec.KAMF(kseaf, supi, abba)

// RES* derivation (TS 33.501 Annex A.4)
resstar, xresstar, err := sec.ResstarXresstar(ckik, servingNetworkName, rand, res)
```

**SQN Resynchronization:**
```go
// When AUTS received from network (sync failure)
sqn, err := m.ValidateAuts(auts, randv)
// sqn contains the recovered sequence number
```

**Handover Key Derivation:**
```go
// Derive AS keys after security mode complete
err := secCtx.DeriveAsKeys()  // Creates KGNB, KN3IWF, NH

// Update NH for handover
err := secCtx.UpdateNh()  // Increments NCC, derives new NH
```

### Key FC Constants (TS 33.501 Annex A)
| Constant | Value | Purpose |
|----------|-------|---------|
| `FC_FOR_KAUSF_DERIVATION` | 0x6A | KAUSF from CK/IK |
| `FC_FOR_KSEAF_DERIVATION` | 0x6C | KSEAF from KAUSF |
| `FC_FOR_KAMF_DERIVATION` | 0x6D | KAMF from KSEAF |
| `FC_FOR_KGNB_KN3IWF_DERIVATION` | 0x6E | RAN keys from KAMF |
| `FC_FOR_NH_DERIVATION` | 0x6F | Next Hop parameter |
| `FC_FOR_RES_STAR_XRES_STAR_DERIVATION` | 0x6B | RES*/XRES* |
| `FC_FOR_ALGORITHM_KEY_DERIVATION` | 0x69 | NAS algorithm keys |
| `FC_FOR_KAMF_PRIME_DERIVATION` | 0x72 | KAMF' for handover/mobility |

### Handover Derivation Parameters
```go
const (
    HDP_NONE          uint8 = iota // No handover
    HDP_HANDOVER                   // Handover (uses DL count)
    HDP_MOBILITY_UPDATE            // Mobility update (uses UL count)
)
```

## Dependencies

### Internal
- `internal/common/logger` - Logging for milenage operations

### External
- `github.com/reogac/nas` - `NasContext` for NAS encryption/integrity
- `github.com/reogac/utils/sec5g` - External 5G security utilities (RanKey, NhKey, KamfPrime)
- `crypto/aes` - AES cipher for Milenage
- `crypto/hmac` - HMAC for KDF
- `crypto/sha256` - SHA-256 for KDF

## Testing
```bash
# Run security package tests
go test ./internal/core/uecontext/sec/...

# Run with verbose output
go test -v ./internal/core/uecontext/sec/...
```

### Test Vectors
- `milenage_test.go`: 5 test cases covering F1-F5* functions with known inputs/outputs
- `ueauth_test.go`: RFC 5448 test vectors for CK'/IK' derivation
