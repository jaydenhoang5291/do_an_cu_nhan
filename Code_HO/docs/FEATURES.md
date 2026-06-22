# StormSim Features and Capabilities

This document outlines the comprehensive 5G procedures, testing capabilities, and advanced features supported by StormSim.

## Supported 5G Procedures

StormSim implements a complete set of 3GPP-compliant 5G procedures for comprehensive network testing.

### 1. TODO:

#### Read [todo note](./TODO.md)

### 2. Registration Management (5GMM)

#### Initial Registration
- **Purpose**: UE first-time network attachment
- **Trigger**: Power-on, manual network selection
- **Sub-procedures**: Authentication, Security Mode Command, Registration Accept
- **Event**: `RegisterInit Event` with `register_type: 0`

#### Deregistration
- **Purpose**: UE detachment from network
- **Types**: 
  - Normal deregistration (`deregister_type: 0`)
  - Switch-off deregistration (`deregister_type: 1`)
- **Event**: `DeregistraterInit Event`

### 3. Session Management (5GSM)

#### PDU Session Establishment
- **Purpose**: Create data connectivity
- **Types**:
  - Initial session (`pdu_session_type: 0`)
  - Emergency session (`pdu_session_type: 1`)
- **Event**: `PduSessionInit Event`

#### PDU Session Modification
- **Purpose**: Update session parameters (QoS, routing rules)
- **Trigger**: Network policy changes, UE requests
- **Event**: `ModificationRequest Event`

#### PDU Session Release
- **Purpose**: Terminate data connectivity
- **Trigger**: UE request, network decision, error conditions
- **Event**: `DestroyPduSession Event`

### 4. Mobility Management

#### Service Request
- **Purpose**: Resume connectivity after idle mode
- **Trigger**: Uplink data transmission, paging response
- **Event**: `ServiceRequestInit Event`

#### Xn Handover
- **Purpose**: Inter-gNB handover via direct interface
- **Requirements**: Source and target gNBs configured
- **Event**: `XnHandover Event`

#### N2 Handover
- **Purpose**: Inter-gNB handover via AMF
- **Use case**: No direct Xn interface between gNBs
- **Event**: `N2Handover Event`

## UE State Machines

### 5GMM State Machine

```
NULL ──► Deregistered ──► Registered
  ▲          │              │
  │          ▼              ▼
  └── DeregInitiated ◄── RegInitiated
```

**States**:
- `NULL State`: Initial state
- `Deregistered State`: Not attached to network
- `DeregistrationInitiated State`: Deregistration in progress
- `AuthenticationInitiated State`: Authentication procedure
- `RegisteredInitiated State`: Registration in progress
- `Registered State`: Successfully attached

### 5GSM State Machine

```
PDUSessionInactive ──► PDUSessionActivePending ──► PDUSessionActive
       ▲                       │                        │
       │                       ▼                        ▼
       └── PDUSessionInactivePending ◄── PDUModificationPending
```

**States**:
- `PDUSessionInactive State`: No active sessions
- `PDUSessionActivePending State`: Session establishment in progress
- `PDUSessionActive State`: Active data session
- `PDUSessionInactivePending State`: Session release in progress
- `PDUModificationPending State`: Session modification in progress

## Authentication and Security

### Supported Authentication Methods
- **5G-AKA**: Primary authentication method
- **EAP-AKA'**: Extended authentication protocol
- **Pre-shared key**: For specific scenarios

### Security Algorithms

#### Integrity Protection (NIA)
- **NIA0**: Null integrity (no protection)
- **NIA1**: SNOW 3G-based integrity
- **NIA2**: AES-based integrity ✅ (Recommended)
- **NIA3**: ZUC-based integrity

#### Ciphering (NEA)
- **NEA0**: Null ciphering (no encryption) ✅ (Testing)
- **NEA1**: SNOW 3G-based encryption
- **NEA2**: AES-based encryption ✅ (Recommended)
- **NEA3**: ZUC-based encryption

### Authentication Parameters
- **IMSI**: International Mobile Subscriber Identity
- **Key (K)**: 128-bit permanent subscription key
- **OPc**: Operator variant algorithm configuration code
- **AMF**: Authentication Management Field
- **SQN**: Sequence Number for replay protection

## Advanced Testing Features

### 1. Fuzzy Testing Engine

**Purpose**: Comprehensive conformance testing and edge case validation

**Capabilities**:
- State machine exploration
- Random event injection
- Protocol boundary testing
- Error condition simulation
- 3GPP compliance validation

**Configuration**:
```yaml
testconf:
  enableFuzz: true
  5gmm:
    states: [registered, deregistered]
    events: [initregister, initderegister, registeraccept, deregisteraccept]
  5gsm:
    states: [pduactive, pduinactive]
    events: [initpdu, acceptpdu, releasepdu]
```

**Output**: Detailed test logs for replay and analysis

### 2. Scenario Recording and Replay

**Recording**: Automatic capture of UE behaviors and network responses
- Message sequences
- Timing information
- State transitions
- Error conditions

**Replay**: Exact reproduction of recorded scenarios
- Deterministic testing
- Regression validation
- Performance comparison
- Issue reproduction

### 3. Multi-UE Group Management

**Capabilities**:
- Independent UE groups with different behaviors
- Scalable from 1 to 10,000+ UEs
- Per-group scenario configuration
- Parallel execution

**Use Cases**:
- Load testing with varied UE behaviors
- Mixed traffic pattern simulation
- Different slice usage patterns
- Staged test execution

### 4. Real-time Remote Control

**REST API Server**:
```yaml
remote:
  enable: true
  ip: "0.0.0.0"
  port: 4000
```

**API Endpoints**:
- `GET /ues` - List all UEs and their states
- `POST /ues/{id}/register` - Trigger UE registration
- `POST /ues/{id}/deregister` - Trigger UE deregistration
- `POST /ues/{id}/session/create` - Establish PDU session
- `POST /ues/{id}/session/release` - Release PDU session
- `POST /ues/{id}/handover` - Initiate handover
- `GET /gnbs` - List gNodeB status
- `GET /metrics` - Retrieve simulation metrics
- `GET /logs` - Access log streams

### 5. Chaos Engineering

**Network Condition Simulation**:
- Packet loss injection
- Latency variation
- Connection drops
- Message corruption
- Resource exhaustion

**Failure Scenarios**:
- SCTP connection failures
- Authentication timeouts
- Protocol violations
- Resource overload
- Recovery testing

### 6. Performance Monitoring

**Built-in Metrics**:
- Registration success rates and latency
- Session establishment times
- Handover completion rates
- Message processing times
- Resource utilization

**External Integration**:
- Prometheus metrics export
- Grafana dashboard support
- Custom metric collectors
- Log aggregation

## UE Behaviors and Event Types

### Lifecycle Events
| Event | Description | Configuration Parameters |
|-------|-------------|-------------------------|
| `RegisterInit Event` | Start registration | `register_type`: 0=Initial, 1=Mobility, 2=Periodic |
| `DeregistraterInit Event` | Start deregistration | `deregister_type`: 0=Normal, 1=Switch-off |
| `ServiceRequestInit Event` | Resume from idle | None |
| `PduSessionInit Event` | Establish session | `pdu_session_type`: 0=Initial, 1=Emergency |
| `DestroyPduSession Event` | Release session | None |

### Mobility Events
| Event | Description | Requirements |
|-------|-------------|--------------|
| `XnHandover Event` | Direct gNB-to-gNB handover | Target gNB configured |
| `N2Handover Event` | AMF-mediated handover | Multiple gNBs, AMF support |

### Control Events
| Event | Description | Use Case |
|-------|-------------|----------|
| `Terminate Event` | Graceful UE shutdown | Clean test completion |
| `Kill Event` | Force UE termination | Error simulation |

## Network Slicing Support

### Slice Configuration
```yaml
defaultUe:
  snssai:
    sst: 01        # Slice/Service Type
    sd: "010203"   # Slice Differentiator (optional)
```

### Supported Slice Types
- **eMBB**: Enhanced Mobile Broadband (SST=1)
- **URLLC**: Ultra-Reliable Low Latency (SST=2)
- **mIoT**: Massive IoT (SST=3)
- **Custom**: Operator-defined slices (SST=128-255)

### Multi-Slice Testing
- Different UE groups per slice
- Slice-specific QoS validation
- Resource isolation testing
- Slice selection accuracy

## Protocol Support

### Interface Support
- **N1**: NAS signaling (UE ↔ AMF)
- **N2**: NGAP signaling (gNB ↔ AMF)
- **N3**: GTP-U data plane (gNB ↔ UPF)
- **Xn**: Inter-gNB signaling

### Message Types
- **NAS**: Registration, Session Management, Security
- **NGAP**: Initial UE Message, Handover procedures
- **GTP-U**: User plane tunneling
- **SCTP**: Reliable transport for signaling

## Quality of Service (QoS)

### QoS Flow Support
- **5QI-based QoS**: Standardized QoS indicators
- **Custom QoS**: Operator-defined parameters
- **Multi-flow sessions**: Multiple QoS flows per PDU session

### QoS Parameters
- **5QI**: 5G QoS Identifier
- **Priority**: Resource scheduling priority
- **Packet Delay Budget**: Maximum acceptable delay
- **Packet Error Rate**: Target error rate
- **Bitrate**: Guaranteed and maximum bitrates

## Scalability and Performance

### UE Scaling
- **Small Scale**: 1-100 UEs for detailed testing
- **Medium Scale**: 100-1,000 UEs for load testing
- **Large Scale**: 1,000-10,000+ UEs for stress testing

### Performance Optimizations
- Concurrent UE processing
- Efficient memory management
- Connection pooling
- Message batching
- State machine optimization

<!-- ### Resource Requirements
| UE Count | CPU Cores | Memory | Network |
|----------|-----------|---------|---------|
| 1-100 | 1-2 | 1-2 GB | 10 Mbps |
| 100-1000 | 2-4 | 2-4 GB | 100 Mbps |
| 1000-5000 | 4-8 | 4-8 GB | 1 Gbps |
| 5000+ | 8+ | 8+ GB | 1+ Gbps | -->

## Integration and Compatibility

### 5G Core Networks
- **Free5GC**: Full compatibility (v3.4.4+)
- **Open5GS**: Full compatibility (latest)
- **OAI 5G-CN**: Basic compatibility
- **Commercial cores**: Protocol-compliant

### Deployment Options
- **Single machine**: Core and emulator co-located
- **Multi-machine**: Distributed deployment
- **Container**: Docker/Kubernetes support
- **Cloud**: Public/private cloud deployment

### Development Integration
- **CI/CD**: Automated testing pipelines
- **Version control**: Git-based configuration management
- **Monitoring**: Integration with observability tools
- **APIs**: External system integration

## Comparison with Other Tools

| Feature | StormSim | UERANSIM | PacketRusher | Commercial Tools |
|---------|-------|----------|--------------|------------------|
| **UE Scale** | 10,000+ | <100 | <1,000 | Varies |
| **Fuzzy Testing** | ✅ | ❌ | ❌ | Limited |
| **Real-time Control** | ✅ | ❌ | ❌ | ✅ |
| **Multi-core Support** | ✅ | ✅ | ✅ | ✅ |
| **Chaos Engineering** | ✅ | ❌ | ❌ | Limited |
| **Open Source** | ✅ | ✅ | ✅ | ❌ |

StormSim distinguishes itself through advanced testing capabilities, high scalability, and comprehensive 5G procedure support, making it ideal for both research and production network validation. 