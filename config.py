"""
Defines global configuration constants for the cellular network simulation,
including radio parameters, handover, stop-and-go mobility, and noise.
"""

# Radio Params
# Based on 3GPP TR 36.777 Table A.1-1
PTX = 46.0       # Power Transmit of Base Station (dBm)
GTX = 2.0        # Gain Transmit
GRX = 0.0        # Gain Receive
FC = 2.0  # GHz
BANDWIDTH = 10e6          # Hz

# Thermal noise model for SINR.
# The receiver noise is independent of the current received signal power
# It is computed from the standard noise density -174 dBm/Hz:
# N_dBm = -174 + 10*log10(B_Hz) + NF_dB
UE_NOISE_FIGURE = 9.0    # dB

HOM = 1.0

# UE speed distribution used by the mobility model.
UE_SPEED_MIN_KMH = 100.0
UE_SPEED_MAX_KMH = 120.0

# Runtime acceleration: evaluate handover candidates among nearby BSs only.
SERVING_BS_CANDIDATES = 12
INTERFERING_BS_COUNT = 6

# Downlink resource grid assumptions for system-level RSRP approximation
# Based on 3GPP TS 36.104 Table 5.6-1 for 10 MHz bandwidth
LTE_N_RB = 50
# Based on 3GPP TS 36.101 Table 6.2.3-1
LTE_N_SUBCARRIERS_PER_RB = 12

H_BS = 25.0     # Height of BS_ground (m)
H_UT = 1.5      # Height of UE_ground (m)
UE_HEIGHT_MIN_M = 1.5
UE_HEIGHT_MAX_M = 300.0

# Shadow Fading
SF_SIGMA = {'LOS': 4.6, 'NLOS': 10.0}
SHADOW_FADING_UPDATE_DISTANCE_M = 25.0

# UE Stop-and-Go config after turns
# Set TURNS_BEFORE_STOP to 0 to disable stop-and-go.
TURNS_BEFORE_STOP = 0
STOP_DURATION_STEPS = 0
RAMP_DURATION_STEPS = 1

# Output sampling
SIMULATION_STEPS = 15000
TIME_PER_STEP_S = 0.02  # 20 ms per simulation step
