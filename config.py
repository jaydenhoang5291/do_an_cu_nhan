"""
Defines global configuration constants for the cellular network simulation,
including radio parameters, handover, stop-and-go mobility, and noise.
"""

# Radio Params
# Based on 3GPP TR 36.777 Table A.1-1
PTX = 46.0       # Power Transmit of Base Station (dBm)
GTX = 2.0        # Gain Transmit
GRX = 2.0        # Gain Receive

SENSITIVITY = -110.0
FC = 2.0  # GHz
HOM = 3.0

MAX_BS_RANGE = 600.0
MAX_CANDIDATE_BS = 6

H_BS = 25.0     # Height of BS_ground (m)
H_UT = 1.5      # Height of UE_ground (m)
UE_HEIGHT_MIN_M = 1.5
UE_HEIGHT_MAX_M = 300.0

# Shadow Fading
SF_SIGMA = {'LOS': 4.6, 'NLOS': 10.0}
SHADOW_FADING_UPDATE_DISTANCE_M = 25.0

# UE Stop-and-Go config after turns
TURNS_BEFORE_STOP = 3
STOP_DURATION_STEPS = 9
RAMP_DURATION_STEPS = 2

# Output sampling
SIMULATION_STEPS = 300
TIME_PER_STEP_S = 1.0

# Noise power
N_DBM = -100.0
