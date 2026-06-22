Now I have deep context on both the codebase and all existing research. Let me run a structured brainstorming session.

---

# FUZZ Testing Brainstorming Session for StormSIM

**Facilitator: Carson -- Brainstorming Coach**

---

## Khung bai toan (Problem Frame)

Dua tren `guide.md` va codebase hien tai, ta co:

| Ky hieu | Y nghia trong StormSIM | Vi du cu the |
|---------|----------------------|-------------|
| **FSM** | `5GMM` (4 states, 12+ events) + `5GSM` (5 states, 9+ events) | `Deregistered -> RegisteredInitiated -> Registered` |
| **ai** (actions) | NAS messages UE gui len / nhan tu Core | `RegistrationRequest`, `AuthenticationResponse`, `SecurityModeComplete`, `UlNasTransport`... |
| **bi** (fields) | Cac truong ben trong NAS message | `MobileIdentity5GS`, `UeSecurityCapability`, `NgKsi`, `SNssai`, `Dnn`, `PduSessionId`... |
| **yi** (fuzz/no-fuzz) | Co mutate truong do hay khong | `yi=1` -> mutate SUCI, `yi=0` -> giu nguyen NgKsi |
| **CheckState()** | Oracle phat hien bug/bat thuong | So sanh response vs 3GPP spec, crash detection, state inconsistency |
| **Special State** | Bug/vulnerability/conformance violation | Core crash, sai state, chap nhan message khong hop le |

---

## ROUND 1: Phan loai bi -- Cac truong NAS quan trong nhat de fuzz

Tu codebase, day la **tat ca bi** co the fuzz, phan loai theo muc do nguy hiem:

### Tier 1: Identity & Authentication (Cao nhat)

| bi | NAS Message | Ly do quan trong |
|----|------------|-----------------|
| `MobileIdentity5GS` (SUCI/GUTI/TMSI) | RegistrationRequest, IdentityResponse | Spoofing, impersonation, IMSI catching |
| `AuthenticationResponseParameter` (RES*) | AuthenticationResponse | Bypass auth, replay attack |
| `AuthenticationFailureParameter` (AUTS) | AuthenticationFailure | SQN desync, DoS |
| `NgKsi` (Key Set Identifier) | Registration, Security, Service | Sai key context, security downgrade |
| `NasMessageContainer` | SecurityModeComplete | Encapsulated reg -> double processing |

### Tier 2: Security Parameters

| bi | NAS Message | Ly do quan trong |
|----|------------|-----------------|
| `UeSecurityCapability` | RegistrationRequest | Force null cipher, downgrade |
| `SelectedNasSecurityAlgorithms` (NEA/NIA) | SecurityModeCommand response | Algorithm mismatch |
| `SecurityHeaderType` | Moi NAS message | Bypass integrity/ciphering |
| `SequenceNumber` | NAS header | Replay, out-of-order |
| `MAC` (integrity) | NAS header | Tampered message acceptance |

### Tier 3: Session & Slice

| bi | NAS Message | Ly do quan trong |
|----|------------|-----------------|
| `SNssai` (SST + SD) | UlNasTransport, RegistrationRequest | Slice isolation bypass |
| `Dnn` | UlNasTransport | Wrong PDN routing |
| `PduSessionId` | UlNasTransport, SM messages | Session hijacking |
| `RequestType` | UlNasTransport | Initial vs existing confusion |
| `PTI` (Procedure Transaction ID) | SM messages | Transaction mismatch |
| `QosRules` / `QosFlowDescriptions` | PDU Session messages | QoS manipulation |

### Tier 4: Procedure & Timing

| bi | NAS Message | Ly do quan trong |
|----|------------|-----------------|
| `RegistrationType` | RegistrationRequest | Initial vs mobility vs emergency |
| `DeRegistrationType` | DeregistrationRequest | Switch-off vs re-registration |
| `UplinkDataStatus` / `PduSessionStatus` | RegistrationRequest | Stale session confusion |
| `5GMMCapability` | RegistrationRequest | Feature negotiation bugs |
| `RequestedNssai` | RegistrationRequest | Slice admission |

---

## ROUND 2: 10 Y tuong FUZZ chinh

### Y tuong 1: **Stateful NAS Mutation Fuzzer** (Co ban, phai co)

**Concept**: Mutate tung truong NAS tai dung thoi diem trong FSM sequence.

```
State: Deregistered
  -> Send RegistrationRequest [mutate bi: SUCI, SecurityCap, Nssai]
  -> Receive AuthenticationRequest
  -> Send AuthenticationResponse [mutate bi: RES*, AUTS]
  -> Receive SecurityModeCommand
  -> Send SecurityModeComplete [mutate bi: NasMessageContainer, NgKsi]
  -> ...
```

**Diem moi so voi hien tai**: `fsm_fuzz.go` hien chi random state/event, KHONG mutate NAS fields. Y tuong nay mutate **noi dung** message, khong chi **thu tu**.

**Implementation hook**: Chen mutator vao giua `trigger*()` va `sendNasToUe()` / `encode()`.

---

### Y tuong 2: **Sequence Permutation Fuzzer** (Thu tu procedure)

**Concept**: Thay doi thu tu cac NAS procedure, pha vo gia dinh cua Core ve happy path.

Vi du:
- Gui `DeregistrationRequest` khi chua `Registered`
- Gui `PduSessionEstablishmentRequest` khi chua xong Registration
- Gui `ServiceRequest` khi dang trong `AuthenticationInitiated`
- Gui hai `RegistrationRequest` lien tiep khong doi response

**Diem manh**: Kiem tra Core co xu ly dung out-of-order messages khong (3GPP clause 5.4.1.7 -- abnormal cases).

**Lien ket FSM**: Day chinh la nhung transition **khong co trong bang** `initFSM()` -- nhung Core van phai xu ly!

---

### Y tuong 3: **Security Context Confusion Fuzzer**

**Concept**: Tap trung vao attack surface lon nhat -- security context.

3 chieu tan cong:
1. **Downgrade**: Gui `UeSecurityCapability` chi co null algorithms (NEA0/NIA0)
2. **Replay**: Gui lai NAS message cu voi `SequenceNumber` cu / MAC cu
3. **Key mismatch**: Dung sai `NgKsi`, tao `AuthenticationResponse` voi RES* tu key khac

**CheckState**: Core PHAI reject. Neu accept -> conformance violation.

**Lien ket code**: `sec.SecurityContext`, `NasContext.DeriveKeys()`, `auth.processAuthenticationInfo()` -- mutate output truoc khi gui.

---

### Y tuong 4: **Slice Isolation Fuzzer** (Multi-UE)

**Concept**: Dung nhieu UE de kiem tra slice isolation.

```
UE1: Register voi Slice A (SST=1, SD=000001)
UE2: Register voi Slice B (SST=1, SD=000002)

FUZZ:
  UE1 gui PduSessionRequest voi SNssai cua Slice B
  -> Core phai reject
  
  UE2 gui UlNasTransport voi PduSessionId cua UE1
  -> Core phai reject
```

**Day la 60% "special state" can 2+ agent** tu guide.md! Coordination workers chuyen lam viec nay.

**CheckState**: Cross-check: UE1 khong the access tai nguyen cua slice khac.

---

### Y tuong 5: **Grammar-Aware IE Mutation Engine**

**Concept**: Thay vi random byte, mutate theo **cau truc IE** (Information Element).

Moi NAS IE co type constraint:
- `MobileIdentity`: SUCI (scheme 0/1), GUTI (format 5G-GUTI), TMSI
- `SNssai`: SST (1 byte, 0-255), SD (3 bytes, optional)
- `UeSecurityCapability`: bitmask cua algorithms

Mutation operators (tu CovFUZZ + AMFuzz):
| Operator | Mo ta | Vi du |
|----------|-------|-------|
| `RAND` | Random value trong type range | SST = random(0,255) |
| `BOUNDARY` | Gia tri bien | PduSessionId = 0, 15, 255 |
| `SWAP` | Hoan doi IE giua hai message | Doi GUTI cua UE1 va UE2 |
| `DELETE` | Bo IE bat buoc | Bo UeSecurityCapability |
| `DUPLICATE` | Lap lai IE | Hai Nssai trong cung request |
| `TYPE_CONFUSE` | Doi type ID nhung giu data | Gui SUCI format nhung dat type = GUTI |
| `OVERFLOW` | Length > actual | SNssai length = 100 |

---

### Y tuong 6: **Coverage-Guided Feedback Loop**

**Concept**: Thay vi fuzz mu, dung feedback tu Core de huong dan mutation.

3 nguon feedback:
1. **State coverage**: Dem so FSM state cua Core da dat (qua NAS response)
2. **Response diversity**: Reject cause codes moi = interesting
3. **Crash/anomaly**: Core restart, timeout, connection drop

```
Loop:
  1. Pick seed (NAS sequence + field config)
  2. Mutate subset bi voi yi=1
  3. Execute sequence
  4. Observe response -> compute coverage score
  5. If new coverage: add to corpus
  6. Update bandit score cho tung bi
```

**Lien ket guide.md**: Day chinh la Contextual Thompson Sampling cho chon bi!

---

### Y tuong 7: **Replay + Differential Oracle**

**Concept**: Chay cung NAS sequence tren **nhieu Core** (Open5GS, free5GC, OAI) va so sanh.

```
Sequence S = [RegReq, AuthResp, SecComplete, PduReq]
Field config = {SUCI: valid, NgKsi: 0, SNssai: (1, "000001")}

Open5GS: Accept
free5GC: Accept  
OAI:     Reject (cause: #11 PLMN not allowed)

-> Differential bug: OAI xu ly PLMN check khac
```

**CheckState differential**:
```
if responses_differ(Open5GS, free5GC, OAI):
    flag_as_potential_bug()
    classify(conformance_vs_implementation)
```

**Lien ket code**: `replay.go` da co mechanism capture/replay -- mo rong de multi-target.

---

### Y tuong 8: **Temporal Race Condition Fuzzer**

**Concept**: Fuzz **timing** giua cac message, khong chi noi dung.

Race conditions:
1. **Concurrent Registration**: 2 UE cung SUPI register dong thoi
2. **Timer race**: Gui message ngay khi timer T3510/T3502 het
3. **Paging + Registration**: UE dang paged nhung cung luc gui Registration
4. **Handover mid-session**: Trigger N2 Handover giua Authentication va SecurityMode

**Lien ket code**: StormSIM da co 100+ worker parallel -- day la loi the lon! `pool.go` co the dieu phoi timing.

**CheckState**: Core phai handle gracefully, khong crash, khong duplicate context.

---

### Y tuong 9: **Conformance Spec Oracle (3GPP-based CheckState)**

**Concept**: Xay dung CheckState() tu 3GPP TS 24.501 / TS 23.502.

Map tu 3GPP spec:

| Spec Clause | Rule | CheckState |
|-------------|------|------------|
| 5.4.1.2.2 | Reg request with invalid SUCI -> reject #3 | `assert response.cause == 3` |
| 5.4.2.4 | Auth failure with MAC fail -> reject cause #20 | `assert response == AuthReject` |
| 5.4.3.3 | Security mode fail -> abort procedure | `assert state == Deregistered` |
| 5.7.1.1 | PDU in wrong slice -> reject #28 | `assert response.cause == 28` |
| 5.6.1.5 | Service req when not registered -> ignore | `assert no_response()` |

**Day la oracle MANH NHAT** vi dua tren dac ta, khong chi crash.

---

### Y tuong 10: **Adaptive Swarm Fuzzer** (Ket hop tat ca)

**Concept**: Ket hop guide.md + tat ca y tuong tren thanh he thong hoan chinh.

```
                    ┌─────────────────┐
                    │  Shared Pool     │
                    │  - bi scores     │
                    │  - path archive  │
                    │  - motifs        │
                    └────────┬────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
     ┌──────▼──────┐ ┌──────▼──────┐ ┌───────▼──────┐
     │ Exploit (45) │ │ Explore (20)│ │ Coord (20)   │
     │ MCTS + known │ │ Random +    │ │ Multi-UE     │
     │ good paths   │ │ Novelty     │ │ Slice/Race   │
     └──────────────┘ └─────────────┘ └──────────────┘
                             │
                    ┌────────▼────────┐
                    │ Anchor (15)     │
                    │ Baseline/valid  │
                    │ Compare         │
                    └─────────────────┘
```

Worker roles:
- **Exploit**: Dung MCTS di theo path da tim duoc signal, mutate nhe
- **Explore**: Random sequence + novel bi combinations
- **Coordination**: Multi-UE scenarios (slice, race, handover)
- **Anchor**: Chay valid sequences lam baseline de detect regression

---

## ROUND 3: Mapping vao code hien tai

| Component | Hien tai | Can xay |
|-----------|---------|---------|
| FSM | `fsm.go` + `fsm_fuzz.go` (random state/event) | **Giu FSM**, them mutation layer truoc NAS encode |
| NAS encode | `trigger.go` builds NAS directly | Them **hook** giua build va encode de mutate fields |
| CheckState | Khong co | Xay module moi: `internal/fuzz/oracle/` |
| bi selection | Khong co | `internal/fuzz/bandit/` -- Thompson Sampling |
| Mutation | Khong co | `internal/fuzz/mutator/` -- grammar-aware |
| Coverage | Khong co | `internal/fuzz/feedback/` -- response analysis |
| Shared Pool | Khong co | `internal/fuzz/pool/` -- kho tri thuc chung |
| Orchestration | `pool.go` workers | Mo rong voi worker roles |
| Capture | `replay.go` co san | Mo rong de luu corpus + differential |

---

## ROUND 4: Priority Ranking

| # | Y tuong | Kho khan | Gia tri | Uu tien |
|---|---------|---------|---------|---------|
| 1 | Stateful NAS Mutation | Trung binh | Rat cao | **P0** |
| 5 | Grammar-Aware IE Mutation | Trung binh | Rat cao | **P0** |
| 9 | Conformance Spec Oracle | Cao | Rat cao | **P0** |
| 2 | Sequence Permutation | Thap | Cao | **P1** |
| 3 | Security Context Confusion | Trung binh | Cao | **P1** |
| 6 | Coverage-Guided Feedback | Cao | Cao | **P1** |
| 4 | Slice Isolation (Multi-UE) | Cao | Rat cao | **P2** |
| 8 | Temporal Race Condition | Cao | Cao | **P2** |
| 7 | Replay + Differential | Trung binh | Trung binh | **P2** |
| 10 | Adaptive Swarm | Rat cao | Rat cao | **P3** (long-term) |

---

## ROUND 5: Concrete bi Table (Toan bo truong co the fuzz)

Day la **bang bi hoan chinh** map tu codebase:

| ID | bi name | NAS IE | Type | Range | Mutation strategy |
|----|---------|--------|------|-------|------------------|
| b1 | SUCI | MobileIdentity5GS | struct | scheme 0/1, MSIN | type_confuse, rand |
| b2 | GUTI | MobileIdentity5GS | struct | 5G-GUTI format | swap, boundary |
| b3 | NgKsi | KeySetIdentifier | uint8 | 0-6, 7=no key | boundary(7), rand |
| b4 | UeSecurityCap | UeSecurityCapability | bitmask | NEA0-3, NIA0-3 | zero_all, single_bit |
| b5 | RegistrationType | 5GSRegistrationType | uint8 | initial/mobility/emergency | type_confuse |
| b6 | SNssai_SST | SNssai | uint8 | 0-255 | boundary, rand |
| b7 | SNssai_SD | SNssai | 3 bytes | optional | delete, overflow |
| b8 | DNN | Dnn | string | APN format | empty, overflow, special_chars |
| b9 | PduSessionId | PduSessionIdentity | uint8 | 1-15 | 0, 16, 255 |
| b10 | RequestType | RequestType | uint8 | initial/existing/mod/emergency | type_confuse |
| b11 | PTI | ProcedureTransactionIdentity | uint8 | 1-254 | 0, 255 |
| b12 | SecurityHeader | SecurityHeaderType | uint4 | 0-4 | 0(plain), rand |
| b13 | SequenceNumber | NAS SN | uint8 | 0-255 | replay(old), skip |
| b14 | MAC | Message Auth Code | 4 bytes | computed | zero, rand, old |
| b15 | NasContainer | NasMessageContainer | bytes | encapsulated PDU | truncate, corrupt |
| b16 | AuthRES | AuthenticationResponseParam | 16 bytes | f2(RAND,K) | zero, rand, partial |
| b17 | AUTS | AuthenticationFailureParam | 14 bytes | f1*(SQN,K) | zero, rand |
| b18 | DeregType | DeRegistrationType | uint8 | normal/switch-off/re-reg | type_confuse |
| b19 | UplinkDataStatus | UplinkDataStatus | bitmask | per-PSI | stale, all_set |
| b20 | PduSessionStatus | PduSessionStatus | bitmask | per-PSI | stale, all_set |
| b21 | RequestedNssai | NSSAI | list[SNssai] | multiple slices | empty, too_many, invalid |
| b22 | 5GMMCapability | 5GMMCapability | bitmask | features | zero, all_set |
| b23 | PayloadContainerType | PayloadContainerType | uint8 | N1SM=1 | wrong_type |
| b24 | SessionType | PDUSessionType | uint8 | IPv4/6/v4v6/unstructured/ethernet | type_confuse |

---

## Ket luan & Buoc tiep theo

Brainstorming session nay da xac dinh:

1. **24 truong bi** cu the tu codebase co the fuzz
2. **10 y tuong fuzz** tu co ban den nang cao
3. **Priority map** -- bat dau tu P0: Stateful NAS Mutation + Grammar-Aware Mutator + Spec Oracle
4. **Architecture** map vao cac package moi can tao
5. **Lien ket** voi guide.md: MCTS, Thompson Sampling, Novelty Search, Multi-Agent Coordination deu co cho trong thiet ke

**Goi y buoc tiep**: Chon 1 trong cac huong sau:
- **A)** Thiet ke chi tiet **architecture** cho fuzz engine (data structures, interfaces, pipeline)
- **B)** Bat tay **implement P0** -- NAS mutation hook + grammar-aware mutator
- **C)** Xay dung **CheckState/Oracle** module tu 3GPP spec truoc
- **D)** Lam **creative squad** session de phat trien them innovation strategy + problem solving

Ban muon di tiep huong nao?
