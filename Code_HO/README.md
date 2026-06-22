# StormSIM HO - 5G Handover Emulator

5G UE and gNodeB emulator for testing and benchmarking 5G Core networks with measurement-based handover support (TN/NTN).

## Prerequisites

- **Go** 1.22.5+
- **Python 3** with `pandas` (`pip install -r requirements.txt`)
- **5G Core** (Open5GS, Free5GC, etc.) running and reachable
- **Root/Sudo** (required for SCTP sockets)

## Quick Start - Handover Monitor Test

### Step 1: Generate RSRP Measurement Data

```bash
python3 data/data.py
```

This generates `rsrp_5g_handover_data_v3.csv` containing:
- 100 time steps of simulated RSRP measurements across 5 gNBs (3 terrestrial + 2 NTN)
- ~17 handover events (including 3 TN-to-NTN transitions)
- Per-step RSRP delta capped at 8 dB with handover margin of 4 dB

### Step 2: Run the Emulator

```bash
go run cmd/emulator/emulator.go -c config/config_lan.yml --csv-measurement rsrp_5g_handover_data_v3.csv --step-delay 100
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-c config/config_lan.yml` | Config file with 5 gNBs and AMF endpoint |
| `--csv-measurement` | Path to RSRP measurement CSV for handover decisions |
| `--step-delay 100` | Delay between handover steps in ms (default: 1000) |

### Configuration

Edit `config/config_lan.yml` to match your network:

```yaml
gnodeb:
  controlif:
    ip: "192.168.1.10"     # N2 interface IP
    port: 9487
  dataif:
    ip: "192.168.1.10"     # N3 interface IP
    port: 2152
  listGnbs:                # 5 gNBs for handover (3 TN + 2 NTN)
    - gnbid: "000001"      # Terrestrial
    - gnbid: "000002"      # Terrestrial
    - gnbid: "000003"      # Terrestrial
    - gnbid: "000004"      # NTN (Non-Terrestrial Network)
    - gnbid: "000005"      # NTN

amfif:
  - ip: "192.168.1.110"   # AMF IP address
    port: 38412
```

### CSV Data Format

The `rsrp_5g_handover_data_v3.csv` file contains these columns:

| Column | Description |
|--------|-------------|
| `Step` | Time step number |
| `gnb1_rsrp` ... `gnb5_rsrp` | RSRP value (dBm) for each gNB |
| `connected_gnb` | Currently serving gNB ID |
| `ue0_handover` | 1 if handover occurs this step, 0 otherwise |
| `ue0_handover_to_type` | Target gNB ID for handover |
| `Type` | Handover type (Xn) |

## Build from Source

```bash
make
```
Binaries output to `bin/emulator` and `bin/client`.

