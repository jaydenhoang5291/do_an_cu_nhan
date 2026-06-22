# StormSim Testing Guide

This guide covers comprehensive testing scenarios for benchmarking 5G core networks using StormSim.

## Testing Overview

StormSim supports various testing methodologies from basic conformance testing to large-scale performance benchmarking. All tests are configured through YAML files and can be executed with simple commands.

## Test Categories

### 1. Conformance Testing
- 3GPP procedure compliance
- Single UE behavior validation
- Fuzzy testing for edge cases

### 2. Performance Testing
- Load testing with multiple UEs
- Throughput and latency measurement
- Resource utilization monitoring

### 3. Reliability Testing
- Handover scenarios
- Chaos injection testing
- Long-duration endurance tests

### 4. Integration Testing
- Multi-core compatibility
- API functionality
- Scenario replay

## Core Network Compatibility

### Supported 5G Cores

| Core | Status | Version Tested | Notes |
|------|--------|----------------|-------|
| **Free5GC** | ✅ Fully Supported | v4.0.1 | Primary development target |
| **Open5GS** | ✅ Fully Supported | Latest | Comprehensive testing |
| **OAI** | ⚠️ Basic Support | 5G-CN | Limited testing |

### StormSim vs Other Tools

| Feature | StormSim | UERANSIM | PacketRusher |
|---------|-------|----------|--------------|
| Multi-UE Scale (>1000) | ✅ | ❌ | ❌ |
| Interactive Control | ✅ | ❌ | ❌ |
| Chaos Injection | ✅ | ❌ | ❌ |
| Fuzzy Testing | ✅ | ❌ | ❌ |
| Handover (Multi-UE) | ✅ | ✅ (Single) | ✅ |
| Performance Metrics | ✅ | Limited | Limited |

## Test Scenarios

### 1. Basic Conformance Tests

#### Single UE Lifecycle Test
**Purpose**: Validate complete UE behavior and 3GPP compliance

```yaml
scenarios:
  - nUEs: 1
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
        register_type: 0
      - event: "PduSessionInit Event"
        delay: 1
        pdu_session_type: 0
      - event: "DestroyPduSession Event"
        delay: 5
      - event: "DeregistraterInit Event"
        delay: 7
        deregister_type: 0
```

**Expected Results**:
- Successful registration within 2-5 seconds
- PDU session establishment within 1-3 seconds
- Clean session release and deregistration
- No protocol errors in core logs

#### Fuzzy Conformance Test
**Purpose**: Test edge cases and protocol robustness

```yaml
testconf:
  enableFuzz: false
  5gmm:
    states:
    - "Registered State"
    - "Deregisterd State"
    events:
    - "RegisterInit Event"
    - "DeregistraterInit Event"
    - "RegistrationAcceptEvent Event"
    - "DeregistrationAcceptEvent Event"
  5gsm:
    states:
    - "PDUSessionActive State"
    - "PDUSessionInactive State"
    events:
    - "PduSessionInit Event"
    - "EstablishmentAccept Event"
    - "DestroyPduSession Event"
```

**Expected Results**:
- All state transitions tested
- Protocol conformance validation
- Replay log generated for analysis

### 2. Performance and Load Tests

#### UE Registration Scalability Test
**Purpose**: Measure maximum registration throughput

```yaml
scenarios:
  - nUEs: 100   # Start with 100, scale up: 200, 500, 1000, 2000, 5000
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
        register_type: 0
```

**Test Progression**:
1. 100 UEs → 200 UEs → 500 UEs → 1000 UEs → 2000 UEs → 5000 UEs → 10000 UEs
2. Monitor core CPU/memory usage
3. Measure registration success rate and latency

**Evaluate to Monitor**:
- Registration latency (target: <2s per UE)
- Success rate (target: >99%)
- Core CPU utilization
- Memory consumption
- Network interface utilization

#### PDU Session Establishment Throughput
**Purpose**: Test session setup performance under load

```yaml
scenarios:
  - nUEs: 1000
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 1
```

**Evaluate**:
- Session setup latency
- Concurrent session capacity
- UPF performance metrics
- NGAP/NAS signaling efficiency

#### Multi-UE Data Plane Stress Test
**Purpose**: Evaluate data throughput with multiple UEs
*Note: Execute this test last as it's most resource-intensive*

```yaml
scenarios:
  - nUEs: 1000
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 1
      # Data traffic generated automatically after session establishment
```

**Evaluate**:
- Aggregate data throughput
- Per-UE throughput consistency
- Packet loss rate
- Jitter and latency
- UPF resource utilization

### 3. Mobility and Handover Tests

#### Xn Handover Test
**Purpose**: Test inter-gNB handover via Xn interface

```yaml
gnodeb:
  listGnbs:
    - gnbid: "000008"
      tac: "000001"
      plmn: {mcc: "208", mnc: "93"}
    - gnbid: "000009"
      tac: "000001"
      plmn: {mcc: "208", mnc: "93"}

scenarios:
  - nUEs: 10
    gnbs: ["000008", "000009"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 2
      - event: "XnHandover Event"
        delay: 5
      - event: "DeregistraterInit Event"
        delay: 10
```

**Evaluate**:
- Handover completion time (target: <500ms)
- Session continuity during handover
- Handover success rate (target: >95%)
- Data interruption time

#### N2 Handover Test
**Purpose**: Test handover via AMF (N2 interface)

```yaml
scenarios:
  - nUEs: 5
    gnbs: ["000008", "000009"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 2
      - event: "N2Handover Event"
        delay: 5
```

### 4. Reliability and Chaos Tests

#### Resiliency Under Abnormal Behavior
**Purpose**: Test core stability under adverse conditions

**Test Configurations**:

1. **Signaling Storm Test**:
```yaml
scenarios:
  - nUEs: 500
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "DeregistraterInit Event"
        delay: 1
      - event: "RegisterInit Event"
        delay: 2  # Rapid re-registration
```

2. **Connection Drop Simulation**:
- Enable chaos injection via test configuration
- Simulate random SCTP connection drops
- Monitor core recovery mechanisms

**Evaluate**:
- System stability during stress
- Recovery time after failures
- Memory leak detection
- Core crash frequency
- Message queue overflow incidents

#### Long-Duration Endurance Test
**Purpose**: Identify long-term stability issues

```yaml
scenarios:
  - nUEs: 200
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
      - event: "PduSessionInit Event"
        delay: 5
      - event: "XnHandover Event"
        delay: 300    # 5 minutes
      - event: "DeregistraterInit Event"
        delay: 600    # 10 minutes
      - event: "RegisterInit Event"
        delay: 900    # Cycle repeats
```

**Duration**: Run for 24-48 hours
**Evaluate**:
- Memory usage trends
- Core log error frequency
- System uptime
- Performance degradation over time

### 5. Network Slicing Tests

#### Slice Selection and Isolation
**Purpose**: Validate network slicing functionality

```yaml
# Configure multiple slices
defaultUe:
  snssai:
    sst: 01
    sd: "010203"

scenarios:
  - nUEs: 50
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
  # Additional scenario groups with different slice configurations
```

**Evaluate**:
- Slice-specific resource allocation
- Traffic isolation between slices
- Slice selection accuracy
- QoS enforcement per slice

## Test Execution Guidelines

### Environment Setup

1. **Recommended VM Configuration**:
   - Core Network: 192.168.56.101 (4 CPU, 8GB RAM)
   - StormSim: 192.168.56.102 (2 CPU, 4GB RAM)

2. **Network Performance Tuning**:
```bash
# Apply before large-scale tests
sudo sysctl -w net.core.rmem_max=33554432
sudo sysctl -w net.core.wmem_max=33554432
sudo sysctl -w net.ipv4.tcp_rmem='4096 65536 33554432'
sudo sysctl -w net.ipv4.tcp_wmem='4096 65536 33554432'
sudo sysctl -w net.core.somaxconn=10240
sudo sysctl -w net.core.netdev_max_backlog=10000
```

### Test Execution Commands

```bash
# Basic conformance test
sudo ./bin/stormsim -c config/conformance.yml

# Load testing progression
sudo ./bin/stormsim -c config/load-100.yml
sudo ./bin/stormsim -c config/load-500.yml
sudo ./bin/stormsim -c config/load-1000.yml

# Handover testing
sudo ./bin/stormsim -c config/handover.yml

# Fuzzy testing
sudo ./bin/stormsim -c config/fuzzy.yml

# Replay test
sudo ./bin/stormsim -r replay_log.yaml -c config/base.yml
```

### Metrics Collection

#### Built-in Metrics

StormSim provides comprehensive metrics during test execution:

| Metric Category | Examples |
|----------------|----------|
| **UE State** | Registration status, session count, handover success |
| **Timing** | Registration latency, session setup time, handover duration |
| **Network** | SCTP connections, NGAP messages, GTP tunnels |
| **Errors** | Authentication failures, protocol errors, timeouts |

#### External Monitoring

**Core Network Monitoring**:
- CPU/Memory usage (`htop`, `free`)
- Network statistics (`ss`, `netstat`)
- Core-specific logs and metrics

**Sample Metrics Table**:

| State | Msg Recv | Time | Event | Start Event | Done Event | Msg Send |
|:-----:|:--------:|:----:|:-----:|:-----------:|:----------:|:--------:|
| deregistered | - | - | register init | 0:31:000004 | - | - |
| deregistered | identity req | 0:31:000014 | - | - | - | identity resp |
| registered | registration accept | 0:31:000034 | - | - | 0:31:000034 | - |

## Test Result Analysis

### Success Criteria

#### Conformance Tests
- ✅ All 3GPP procedures complete successfully
- ✅ No protocol violations in logs
- ✅ State transitions follow specifications
- ✅ Authentication and security procedures work correctly

#### Performance Tests
- ✅ Target UE count achieved without failures
- ✅ Registration latency < 5 seconds at scale
- ✅ >95% success rate for all procedures
- ✅ Core resource utilization < 80%

#### Reliability Tests
- ✅ System remains stable under stress
- ✅ Recovery time < 30 seconds after failures
- ✅ No memory leaks over 24+ hour runs
- ✅ Handover success rate > 90%

### Common Issues and Solutions

#### Performance Issues
1. **High latency at scale**:
   - Apply network tuning parameters
   - Increase core resources
   - Reduce concurrent UE ramp rate

2. **Authentication failures**:
   - Verify UE database provisioning
   - Check key/OPC configuration
   - Validate PLMN settings

3. **Connection failures**:
   - Verify SCTP module loaded
   - Check firewall rules
   - Validate IP addressing

#### Test-Specific Issues
1. **Handover failures**:
   - Ensure multiple gNBs configured
   - Verify gNB neighbor relationships
   - Check TAC/PLMN consistency

2. **Fuzzy test issues**:
   - Only use single UE for fuzzy testing
   - Check state machine configurations
   - Review generated logs for insights

## Continuous Integration

### Automated Test Suites

```bash
#!/bin/bash
# Basic CI test suite

# Conformance tests
sudo ./bin/stormsim -c config/ci/single-ue.yml
sudo ./bin/stormsim -c config/ci/fuzzy.yml

# Load tests
sudo ./bin/stormsim -c config/ci/load-100.yml
sudo ./bin/stormsim -c config/ci/load-500.yml

# Handover tests
sudo ./bin/stormsim -c config/ci/handover.yml
```

### Test Reporting

Generate comprehensive test reports including:
- Test execution summary
- Performance metrics
- Error analysis
- Core network resource utilization
- Recommendations for optimization

## Advanced Testing Features

### API Testing

When remote API is enabled:
```bash
# Test API endpoints
curl http://localhost:4000/ues
curl -X POST http://localhost:4000/ues/1/register
curl http://localhost:4000/metrics
```

### Custom Scenario Development

Create custom test scenarios by:
1. Defining UE group behaviors
2. Configuring event sequences
3. Setting timing parameters
4. Enabling specific monitoring

This comprehensive testing approach ensures thorough validation of 5G core networks across conformance, performance, and reliability dimensions. 
