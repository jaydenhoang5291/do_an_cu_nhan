# StormSim Configuration Guide

This guide provides detailed information on configuring StormSim for various testing scenarios.

## Overview

StormSim is entirely configuration-driven through YAML files. The main configuration file is `config/config.yml`, which defines UE behaviors, gNB parameters, scenarios, and testing options.

## Configuration File Structure

```yaml
gnodeb:           # gNodeB configuration
defaultUe:        # Default UE profile
amfif:           # AMF interface endpoints
scenarios:       # Test scenarios and UE behaviors
remote:          # Remote API server settings
testconf:        # Testing and fuzzing options
loglevel:        # Logging configuration
```

## Detailed Configuration Sections

### 1. gNodeB Configuration (`gnodeb`)

Configure gNodeB parameters including interfaces and supported slices.

```yaml
gnodeb:
  controlif:
    ip: "127.0.0.20"        # N2 interface IP for NGAP/NAS signaling
    port: 9487              # N2 interface port
  dataif:
    ip: "127.0.0.20"        # N3 interface IP for GTP-U traffic
    port: 2152              # N3 interface port (standard GTP-U port)
  listGnbs:                 # Multiple gNBs for handover scenarios
    - gnbid: "000008"       # Unique gNodeB identifier
      tac: "000001"         # Tracking Area Code
      plmn:
        mcc: "208"          # Mobile Country Code
        mnc: "93"           # Mobile Network Code
      slicesupportlist:
        - sst: "01"         # Slice/Service Type
          sd: "010203"      # Slice Differentiator (optional)
```

**Interface Parameters:**
- **controlif**: N2 interface for AMF communication (SCTP)
- **dataif**: N3 interface for UPF communication (UDP)
- **Multiple gNBs**: Required for handover testing (Xn/N2)

### 2. Default UE Profile (`defaultUe`)

Configure authentication parameters and network settings for UEs.

```yaml
defaultUe:
  msin: "0000000000"                           # Mobile Subscriber ID Number
  key: "14b23ceb27e95eb732a3f9d602f551c4"      # 128-bit authentication key
  opc: "a3e3c63de23b66dc6a8ae0272b44906c"      # Operator Code
  amf: "8000"                                  # Authentication Management Field
  sqn: "00000000"                              # Sequence Number
  dnn: "internet"                              # Data Network Name
  routingindicator: "0000"                     # Routing Indicator
  hplmn:                                       # Home PLMN
    mcc: "208"
    mnc: "93"
  snssai:                                      # Network Slice Selection
    sst: 01
    sd: "010203"
  integrity:                                   # Integrity algorithms
    nia0: false    # Null integrity
    nia1: false    # SNOW 3G
    nia2: true     # AES (recommended)
    nia3: false    # ZUC
  ciphering:                                   # Ciphering algorithms
    nea0: true     # Null ciphering
    nea1: false    # SNOW 3G
    nea2: true     # AES (recommended)
    nea3: false    # ZUC
```

**Authentication Parameters:**
- **key**: Permanent subscription key (must match 5G core provisioning)
- **opc**: Operator variant algorithm configuration code
- **msin**: Auto-incremented for multiple UEs (0000000001, 0000000002, etc.)

### 3. AMF Interfaces (`amfif`)

Configure AMF endpoints for core network connectivity.

```yaml
amfif:
  - ip: "127.0.0.8"         # AMF N2 interface IP
    port: 38412             # AMF N2 interface port (standard NGAP port)
  - ip: "192.168.1.100"     # Additional AMF for redundancy/handover
    port: 38412
```

**Multiple AMFs**: Support multiple AMFs for failover and inter-AMF handover scenarios.

### 4. Test Scenarios (`scenarios`)

Define UE groups, behaviors, and event sequences.

```yaml
scenarios:
  - nUEs: 100               # Number of UEs in this group
    gnbs: ["000008"]        # gNB IDs this UE group will use
    ueEvents:               # Sequence of events for UEs
      - event: "RegisterInit Event"
        delay: 0            # Start immediately
        register_type: 0    # Initial registration
      - event: "PduSessionInit Event"
        delay: 2            # 2 seconds after previous event
        pdu_session_type: 0 # Initial session
      - event: "DeregistraterInit Event"
        delay: 10           # 10 seconds after previous event
        deregister_type: 0  # Normal deregistration
```

#### Available UE Events

| Event | Description | Parameters |
|-------|-------------|------------|
| `RegisterInit Event` | Start registration procedure | `register_type`: 0=Initial, 1=Mobility, 2=Periodic |
| `DeregistraterInit Event` | Start deregistration | `deregister_type`: 0=Normal, 1=Switch-off |
| `ServiceRequestInit Event` | Service request procedure | - |
| `PduSessionInit Event` | Establish PDU session | `pdu_session_type`: 0=Initial, 1=Emergency |
| `DestroyPduSession Event` | Release PDU session | - |
| `XnHandover Event` | Xn interface handover | Target gNB must be configured |
| `N2Handover Event` | N2 interface handover | Target gNB must be configured |
| `Terminate Event` | Graceful UE termination | - |
| `Kill Event` | Force UE termination | - |

#### Multiple UE Groups

```yaml
scenarios:
  - nUEs: 50                # Group 1: Registration only
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
  - nUEs: 30                # Group 2: Full lifecycle
    gnbs: ["000008", "000009"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 1
      - event: "XnHandover Event"
        delay: 5
      - event: "DeregistraterInit Event"
        delay: 10
```

### 5. Remote API Server (`remote`)

Enable REST API for external control and monitoring.

```yaml
remote:
  enable: true              # Enable remote API server
  ip: "0.0.0.0"            # Bind to all interfaces
  port: 4000               # API server port
```

**API Endpoints** (when enabled):
- `GET /ues` - List all UEs and status
- `POST /ues/{id}/register` - Trigger UE registration
- `POST /ues/{id}/deregister` - Trigger UE deregistration
- `GET /gnbs` - List gNodeB status
- `GET /metrics` - Get simulation metrics

### 6. Testing Configuration (`testconf`)

Configure fuzzing and conformance testing options.

```yaml
testconf:
  enableFuzz: true                        # Enable fuzzy testing (single UE only)
  5gmm:                                   # 5G Mobility Management state machine
    states:
    - "Registered State"
    - "Deregisterd State"
    events:
    - "RegisterInit Event"
    - "DeregistraterInit Event"
    - "RegistrationAcceptEvent Event"
    - "DeregistrationAcceptEvent Event"
  5gsm:                                   # 5G Session Management state machine
    states:
    - "PDUSessionActive State"
    - "PDUSessionInactive State"
    events:
    - "PduSessionInit Event"
    - "EstablishmentAccept Event"
    - "DestroyPduSession Event"
```

**Fuzzy Testing:**
- Only works with single UE scenarios
- Generates comprehensive test logs for replay
- Tests all defined state transitions
- Validates 3GPP conformance

### 7. Logging Configuration

```yaml
loglevel: info # Log levels: debug, info, warn, error, fatal, panic
```

## Example Configurations

### Basic Single UE Test
```yaml
scenarios:
  - nUEs: 1
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 1
```

### Load Testing (1000 UEs)
```yaml
scenarios:
  - nUEs: 1000
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
```

### Handover Scenario
```yaml
gnodeb:
  listGnbs:
    - gnbid: "000008"
      tac: "000001"
      plmn: { mcc: "208", mnc: "93" }
      slicesupportlist: [{ sst: "01", sd: "010203" }]
    - gnbid: "000009"         # Second gNB for handover
      tac: "000001"
      plmn: { mcc: "208", mnc: "93" }
      slicesupportlist: [{ sst: "01", sd: "010203" }]

scenarios:
  - nUEs: 10
    gnbs: ["000008", "000009"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 1
      - event: "XnHandover Event"
        delay: 5
```

### Conformance Testing (Fuzzy)
```yaml
testconf:
  enableFuzz: true

scenarios:
  - nUEs: 1                 # Fuzzy testing requires single UE
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
```

## Core Network Integration

### Free5GC Configuration

Ensure your Free5GC configuration matches StormSim parameters:

1. **AMF Configuration** (`config/amfcfg.yaml`):
```yaml
configuration:
  amfName: AMF
  ngapIpList:
    - "127.0.0.8"
  sbi:
    scheme: http
    registerIPv4: 127.0.0.8
```

2. **Add UE subscribers** to Free5GC database with matching authentication parameters.

### Open5GS Configuration

Configure Open5GS AMF to match StormSim gNB interfaces:

```yaml
amf:
  ngap:
    - addr: 127.0.0.8
      port: 38412
```

## Network Setup (NO NEED)

### Single Machine Setup
- Use loopback interfaces (127.0.0.x)
- Core: 127.0.0.8
- StormSim: 127.0.0.20

### Multi-Machine Setup
- Core: 192.168.1.100
- StormSim: 192.168.1.101
- Ensure proper routing and firewall rules

### Performance Tuning

For large-scale tests (>1000 UEs):

```bash
# Increase network buffers
sudo sysctl -w net.core.rmem_max=33554432
sudo sysctl -w net.core.wmem_max=33554432
sudo sysctl -w net.ipv4.tcp_rmem='4096 65536 33554432'
sudo sysctl -w net.ipv4.tcp_wmem='4096 65536 33554432'

# Increase connection limits
sudo sysctl -w net.core.somaxconn=10240
sudo sysctl -w net.core.netdev_max_backlog=10000
```

## Troubleshooting

### Common Issues

1. **SCTP Connection Failed**
   - Verify SCTP kernel module: `sudo modprobe sctp`
   - Check AMF connectivity and firewall rules

2. **Authentication Failures**
   - Verify UE key/opc matches 5G core database
   - Check PLMN (MCC/MNC) configuration

3. **High UE Count Failures**
   - Apply network performance tuning
   - Increase system limits (ulimit -n)
   - Consider reducing concurrent UE count

### Debug Configuration

```yaml
loglevel: debug             # Enable detailed logging
testconf:
  enableFuzz: false         # Disable fuzzing for cleaner logs
```

For detailed troubleshooting, enable debug logging and check both StormSim and 5G core logs for specific error messages. 