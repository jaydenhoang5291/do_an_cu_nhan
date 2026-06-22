import random
import pandas as pd

# =========================================
# CONFIG
# =========================================
NUM_STEPS = 100
NUM_HANDOVERS = 17
NUM_TN_TO_NTN_HANDOVERS = 3
MAX_DELTA_PER_STEP = 8
HANDOVER_MARGIN = 4  # next_bs must be at least this better than all others
OUTPUT_FILE = "rsrp_5g_handover_data_v3.csv"

random.seed(42)

BS_IDS = ["000001", "000002", "000003", "000004", "000005"]
TERRESTRIAL_BS = ["000001", "000002", "000003"]
NTN_BS = ["000004", "000005"]

GLOBAL_RANGE = {
    "000001": (-120, -75),
    "000002": (-120, -75),
    "000003": (-120, -75),
    "000004": (-140, -108),
    "000005": (-155, -100),
}

NORMAL_CENTER_RANGE = {
    "000001": (-90, -80),
    "000002": (-90, -80),
    "000003": (-92, -82),
    "000004": (-118, -112),
    "000005": (-140, -130),
}


# =========================================
# HELPER
# =========================================
def clamp(x, low, high):
    return max(low, min(x, high))


def bounded_step(prev, target, low, high, max_delta=MAX_DELTA_PER_STEP):
    val = prev + max(- max_delta, min(max_delta, target - prev))
    return clamp(val, low, high)


def choose_segment_lengths(total_steps, num_segments, min_len=3):
    if num_segments * min_len > total_steps:
        raise ValueError("Not enough steps to split into segments.")
    lengths = [min_len] * num_segments
    remain = total_steps - sum(lengths)
    for _ in range(remain):
        lengths[random.randint(0, num_segments - 1)] += 1
    return lengths


def choose_special_boundaries(num_segments, num_special):
    boundaries = list(range(1, num_segments))
    size = len(boundaries)
    g1 = boundaries[:max(1, size // 3)]
    g2 = boundaries[max(1, size // 3):max(2, 2 * size // 3)]
    g3 = boundaries[max(2, 2 * size // 3):]
    groups = [g for g in [g1, g2, g3] if g]
    selected = []
    for g in groups:
        if len(selected) < num_special:
            selected.append(random.choice(g))
    remain = [x for x in boundaries if x not in selected]
    while len(selected) < num_special and remain:
        x = random.choice(remain)
        selected.append(x)
        remain.remove(x)
    return sorted(selected)


def generate_connected_sequence(num_steps, num_handovers, num_special):
    num_segments = num_handovers + 1
    seg_lengths = choose_segment_lengths(num_steps, num_segments, min_len=3)
    special_boundaries = choose_special_boundaries(num_segments, num_special)
    segments = [random.choice(TERRESTRIAL_BS)]
    for seg_idx in range(1, num_segments):
        prev_bs = segments[-1]
        if seg_idx in special_boundaries and prev_bs in TERRESTRIAL_BS:
            next_bs = random.choice(NTN_BS)
        else:
            candidates = [x for x in BS_IDS if x != prev_bs]
            next_bs = random.choice(candidates)
        segments.append(next_bs)
    seq = []
    for bs, length in zip(segments, seg_lengths):
        seq.extend([bs] * length)
    return seq[:num_steps]


def get_handover_rows(seq):
    return [i for i in range(len(seq) - 1) if seq[i] != seq[i + 1]]


def is_terrestrial_to_ntn(seq, row_idx):
    return (row_idx < len(seq) - 1
            and seq[row_idx] in TERRESTRIAL_BS
            and seq[row_idx + 1] in NTN_BS)


def init_prev_vals():
    vals = {}
    for bs in BS_IDS:
        lo, hi = NORMAL_CENTER_RANGE[bs]
        vals[bs] = random.randint(lo, hi)
    return vals


def make_target_profile(current_bs, next_bs=None, special=False):
    """Build an ideal RSRP target for this step, ignoring delta limits.
    bounded_step() will enforce the per-step delta later."""
    t = {
        "000001": random.randint(-90, -80),
        "000002": random.randint(-90, -80),
        "000003": random.randint(-92, -82),
        "000004": random.randint(-118, -112),
        "000005": random.randint(-140, -130),
    }

    if next_bs is None:
        # Normal row: serving BS is best
        if current_bs in TERRESTRIAL_BS:
            best = random.randint(-88, -79)
        elif current_bs == 11:
            best = random.randint(-115, -109)
        else:
            best = random.randint(-120, -112)
        t[current_bs] = best
        for bs in BS_IDS:
            if bs != current_bs and t[bs] >= best:
                t[bs] = best - random.randint(1, 6)
    else:
        # Handover row: next_bs must be best by HANDOVER_MARGIN
        if special and current_bs in TERRESTRIAL_BS and next_bs in NTN_BS:
            # TN->NTN: push current TN down hard, lift NTN target
            t[current_bs] = random.randint(-120, -118)
            for bs in TERRESTRIAL_BS:
                if bs != current_bs:
                    t[bs] = random.randint(-120, -108)
            best = random.randint(-111, -104) if next_bs == 11 else random.randint(-113, -104)
            t[next_bs] = best
            for bs in [x for x in NTN_BS if x != next_bs]:
                t[bs] = random.randint(-155, best - HANDOVER_MARGIN)
        else:
            if next_bs in TERRESTRIAL_BS:
                best = random.randint(-86, -78)
            elif next_bs == "000004":
                best = random.randint(-112, -108)
            else:
                best = random.randint(-114, -104)
            t[next_bs] = best
            for bs in BS_IDS:
                if bs != next_bs:
                    low_floor = -155 if bs in NTN_BS else -120
                    t[bs] = random.randint(low_floor, best - HANDOVER_MARGIN)

    for bs in BS_IDS:
        lo, hi = GLOBAL_RANGE[bs]
        t[bs] = clamp(t[bs], lo, hi)
    return t


def smooth_row(prev_vals, target_profile):
    """Move each BS value toward target, respecting MAX_DELTA_PER_STEP."""
    return {
        bs: bounded_step(prev_vals[bs], target_profile[bs], *GLOBAL_RANGE[bs])
        for bs in BS_IDS
    }


def enforce_handover_margin(prev_vals, vals, next_bs):
    """
    Ensure next_bs leads all others by HANDOVER_MARGIN dB.
    All adjustments respect MAX_DELTA_PER_STEP relative to prev_vals.
    Returns (adjusted_vals, success).
    """
    lo_t, hi_t = GLOBAL_RANGE[next_bs]

    # Max reachable value for next_bs given delta constraint
    max_next = clamp(prev_vals[next_bs] + MAX_DELTA_PER_STEP, lo_t, hi_t)

    # Min reachable value for each other BS given delta constraint
    min_other = {
        bs: clamp(prev_vals[bs] - MAX_DELTA_PER_STEP, *GLOBAL_RANGE[bs])
        for bs in BS_IDS if bs != next_bs
    }

    # Check feasibility: can next_bs beat the lowest possible rival?
    worst_rival_min = max(min_other[bs] for bs in min_other)
    if max_next < worst_rival_min + HANDOVER_MARGIN:
        # Physically impossible within delta limits — skip enforcement
        return vals, False

    # Set next_bs as high as delta allows
    vals[next_bs] = max_next

    # Push rivals down as far as needed (within delta)
    threshold = vals[next_bs] - HANDOVER_MARGIN
    for bs in BS_IDS:
        if bs == next_bs:
            continue
        lo, hi = GLOBAL_RANGE[bs]
        if vals[bs] > threshold:
            # clamp to [min reachable, threshold]
            vals[bs] = clamp(threshold, min_other[bs], hi)

    # Final check
    others_max = max(vals[bs] for bs in BS_IDS if bs != next_bs)
    success = vals[next_bs] >= others_max + HANDOVER_MARGIN
    return vals, success


# =========================================
# MAIN
# =========================================
connected_seq = generate_connected_sequence(
    NUM_STEPS, NUM_HANDOVERS, NUM_TN_TO_NTN_HANDOVERS
)

handover_rows = get_handover_rows(connected_seq)
special_rows = {r for r in handover_rows if is_terrestrial_to_ntn(connected_seq, r)}

data = []
prev_vals = init_prev_vals()
enforcement_failures = []

for i in range(NUM_STEPS):
    current_bs = connected_seq[i]
    is_handover = i < NUM_STEPS - 1 and connected_seq[i] != connected_seq[i + 1]

    if is_handover:
        next_bs = connected_seq[i + 1]
        special = i in special_rows
        target_profile = make_target_profile(current_bs, next_bs=next_bs, special=special)
        vals = smooth_row(prev_vals, target_profile)
        vals, ok = enforce_handover_margin(prev_vals, vals, next_bs)
        if not ok:
            enforcement_failures.append(i + 1)
        ue0_handover = 1
        ue0_handover_to_type = next_bs
    else:
        target_profile = make_target_profile(current_bs)
        vals = smooth_row(prev_vals, target_profile)
        ue0_handover = 0
        ue0_handover_to_type = current_bs

    data.append({
        "Step":                 i + 1,
        "gnb1_rsrp":            vals["000001"],
        "gnb2_rsrp":            vals["000002"],
        "gnb3_rsrp":            vals["000003"],
        "gnb4_rsrp":            vals["000004"],
        "gnb5_rsrp":            vals["000005"],
        "connected_gnb":        current_bs,
        "ue0_BS_ketnoi":        current_bs,
        "ue0_handover":         ue0_handover,
        "ue0_handover_to_type": ue0_handover_to_type,
        "Type":                 "Xn",
    })
    prev_vals = vals.copy()

df = pd.DataFrame(data)

# =========================================
# CHECK
# =========================================
print(f"Tổng số handover: {df['ue0_handover'].sum()}")

for col in ["gnb1_rsrp", "gnb2_rsrp", "gnb3_rsrp", "gnb4_rsrp", "gnb5_rsrp"]:
    max_diff = df[col].diff().abs().fillna(0).max()
    print(f"Max delta của {col}: {max_diff}")

valid_logic = True
for i in range(NUM_STEPS - 1):
    if df.loc[i, "ue0_handover"] == 1:
        target = df.loc[i, "ue0_handover_to_type"]
        rsrps = {
            "000001": df.loc[i, "gnb1_rsrp"],
            "000002": df.loc[i, "gnb2_rsrp"],
            "000003": df.loc[i, "gnb3_rsrp"],
            "000004": df.loc[i, "gnb4_rsrp"],
            "000005": df.loc[i, "gnb5_rsrp"],
        }
        target_val = rsrps[target]
        for bs, val in rsrps.items():
            if bs != target and not (target_val - 3 > val):
                valid_logic = False
                print(f"Lỗi handover row {i+1}: target={target}, target_val={target_val}, bs={bs}, val={val}")

if enforcement_failures:
    print(f"Rows where margin enforcement was infeasible (skipped): {enforcement_failures}")
print("Check handover logic:", valid_logic)

df.to_csv(OUTPUT_FILE, index=False)
print(f"Đã lưu file: {OUTPUT_FILE}")
print(df.head(20))
