import csv
import sys
from collections import defaultdict

csv_file = sys.argv[1] if len(sys.argv) > 1 else "timer-events-1781607216.csv"

rows = []
with open(csv_file, newline='') as f:
    reader = csv.DictReader(f)
    for row in reader:
        rows.append(row)

# Group by trial_id + timer_name
groups = defaultdict(list)
for r in rows:
    key = (r['trial_id'], r['source_gnb'], r['target_gnb'])
    groups[key].append(r)

print("=== Timer Event Analysis ===")
print(f"Total events: {len(rows)}")
print(f"Total handover trials: {len(groups)}")
print()

# Analyze TXnRELOCprep failures
print("--- TXnRELOCprep Failure Analysis ---")
prep_fails = []
for key, events in groups.items():
    for e in events:
        if e['timer_name'] == 'TXnRELOCprep' and e['event'] == 'transport_fail':
            prep_fails.append(e)

print(f"Transport failures (xn_transport_fail): {len(prep_fails)}")
for e in prep_fails:
    print(f"  Trial {e['trial_id']}: {e['source_gnb']}->{e['target_gnb']} elapsed={e['elapsed_ms']}ms")

print()

# Analyze TXnRELOCprep timeouts
prep_timeouts = []
for key, events in groups.items():
    for e in events:
        if e['timer_name'] == 'TXnRELOCprep' and e['event'] == 'timeout':
            prep_timeouts.append(e)

print(f"Timer timeouts (natural expiry): {len(prep_timeouts)}")
for e in prep_timeouts:
    src_target = f"{e['source_gnb']}->{e['target_gnb']}"
    had_transport_fail = any(
        x['event'] == 'transport_fail' and x['source_gnb'] == e['source_gnb'] and x['target_gnb'] == e['target_gnb']
        for x in prep_fails
    )
    marker = " [after transport_fail]" if had_transport_fail else " [direct timeout]"
    print(f"  Trial {e['trial_id']}: {src_target} elapsed={e['elapsed_ms']}ms{marker}")

print()

# TXnRELOCprep success events
prep_success = []
for key, events in groups.items():
    for e in events:
        if e['timer_name'] == 'TXnRELOCprep' and e['event'] == 'stop':
            prep_success.append(e)

print(f"TXnRELOCprep success (stop): {len(prep_success)}")

# Summary
total_prep = len(prep_success) + len(prep_fails) + len(prep_timeouts)
# But timeouts that follow transport_fail are counted in both:
# transport_fail trials where prep also timed out
transport_with_timeout = 0
for pf in prep_fails:
    for pt in prep_timeouts:
        if pf['trial_id'] == pt['trial_id'] and pf['start_ts'] == pt['start_ts']:
            transport_with_timeout += 1
            break

print()
print("=== Summary ===")
print(f"Total TXnRELOCprep lifecycles: {total_prep}")
print(f"  - Success (stop): {len(prep_success)}")
print(f"  - Transport failure (xn_transport_fail): {len(prep_fails)}")
print(f"  - Direct timeout (no prior transport_fail): {len(prep_timeouts) - transport_with_timeout}")
print(f"  - Timeout after transport_fail: {transport_with_timeout}")
print(f"Transport failure rate: {len(prep_fails)}/{total_prep} = {100*len(prep_fails)/max(total_prep,1):.1f}%")
print(f"Expected rate with XnPacketLoss=0.5: 12.5% (0.5^3)")
