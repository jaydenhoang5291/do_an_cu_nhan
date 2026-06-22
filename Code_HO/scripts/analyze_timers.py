#!/usr/bin/env python3
import pandas as pd
import sys
from pathlib import Path

import argparse

parser = argparse.ArgumentParser(description='Analyze timer CSV for anomalies')
parser.add_argument('csv', nargs='?', default='timer-events-1781055555.csv', help='CSV file to analyze')
args = parser.parse_args()

p = Path(args.csv)
if not p.exists():
    print('CSV file not found:', p)
    sys.exit(1)

df = pd.read_csv(p, parse_dates=['ts'])
# keep relevant columns
cols = ['ts','trial_id','timer_name','event','elapsed_ms']
for c in cols:
    if c not in df.columns:
        print('Missing column', c)
        sys.exit(1)

df = df[cols].sort_values(['trial_id','timer_name','ts'])

tolerance_ms = 50.0
anomalies = []

# group by trial_id and timer_name
for (trial,timer), g in df.groupby(['trial_id','timer_name']):
    starts = g[g['event']=='start']
    stops = g[g['event'].isin(['stop','timeout'])]
    if len(stops) > 0 and len(starts) == 0:
        for _, row in stops.iterrows():
            anomalies.append({'type':'stop_without_start','trial_id':trial,'timer_name':timer,'ts':row['ts'],'event':row['event'],'elapsed_ms':row['elapsed_ms']})
    # for each stop, find most recent start <= stop.ts
    start_times = list(starts['ts'])
    for _, stop in stops.iterrows():
        last_start = None
        for st in start_times:
            if st <= stop['ts']:
                last_start = st
        if last_start is None:
            anomalies.append({'type':'stop_without_start_ts','trial_id':trial,'timer_name':timer,'ts':stop['ts'],'event':stop['event'],'elapsed_ms':stop['elapsed_ms']})
        else:
            delta_ms = (stop['ts'] - last_start).total_seconds()*1000.0
            # handle NaN elapsed
            try:
                em = float(stop['elapsed_ms'])
            except Exception:
                em = None
            if em is None:
                anomalies.append({'type':'elapsed_missing','trial_id':trial,'timer_name':timer,'ts':stop['ts'],'event':stop['event'],'delta_ms':delta_ms,'elapsed_ms':em})
            else:
                if abs(delta_ms - em) > tolerance_ms:
                    anomalies.append({'type':'elapsed_mismatch','trial_id':trial,'timer_name':timer,'ts':stop['ts'],'event':stop['event'],'delta_ms':delta_ms,'elapsed_ms':em})

# summary
from collections import Counter
cnt = Counter(a['type'] for a in anomalies)
print('Analyzed file:', p)
print('Total rows:', len(df))
print('Anomalies found:', len(anomalies))
for k,v in cnt.items():
    print(f'  {k}: {v}')

# print first 20 anomalies
print('\nFirst anomalies:')
for a in anomalies[:20]:
    print(a)

# write anomalies to a CSV for inspection
out = Path(f'timer_anomalies_{p.stem}.csv')
import csv
# normalize keys across anomalies
fieldnames = set()
for a in anomalies:
    fieldnames.update(a.keys())
fieldnames = list(sorted(fieldnames)) if fieldnames else ['type']
with out.open('w', newline='') as f:
    w = csv.DictWriter(f, fieldnames=fieldnames)
    w.writeheader()
    for a in anomalies:
        # ensure all fields present
        row = {k: a.get(k, '') for k in fieldnames}
        w.writerow(row)
print('\nWrote anomalies to', out)

# also produce a deduplicated summary: unique stop events without a prior start
unique_no_start = []
seen = set()
for a in anomalies:
    if a['type'].startswith('stop_without_start'):
        key = (a['trial_id'], a['timer_name'], a['ts'])
        if key not in seen:
            seen.add(key)
            unique_no_start.append(a)

print('\nUnique stop-without-start examples (up to 20):')
for a in unique_no_start[:20]:
    print(a)
print('\nUnique stop-without-start count:', len(unique_no_start))

# Build paired summary: match starts to subsequent stops (FIFO per trial_id+timer_name)
summary_rows = []
from collections import deque
groups = df.groupby(['trial_id','timer_name'])
for (trial,timer), g in groups:
    q = deque()
    g_sorted = g.sort_values('ts')
    for _, row in g_sorted.iterrows():
        ev = str(row['event']).strip()
        ts = row['ts']
        em = row.get('elapsed_ms', '')
        res = row.get('result', '') if 'result' in row.index else ''
        if ev == 'start':
            q.append({'start_ts': ts, 'start_row': row})
        elif ev in ('stop','timeout'):
            if q:
                st = q.popleft()
                start_ts = st['start_ts']
                stop_ts = ts
                computed_ms = (stop_ts - start_ts).total_seconds()*1000.0
                recorded_ms = None
                try:
                    recorded_ms = float(em)
                except Exception:
                    recorded_ms = None
                expired_flag = True if ev == 'timeout' else False
                summary_rows.append({
                    'trial_id': trial,
                    'timer_name': timer,
                    'start_ts': start_ts.isoformat(),
                    'stop_ts': stop_ts.isoformat(),
                    'recorded_elapsed_ms': recorded_ms,
                    'computed_elapsed_ms': round(computed_ms,3),
                    'event': ev,
                    'result': res,
                    'expired': expired_flag
                })
            else:
                # stop without start
                recorded_ms = None
                try:
                    recorded_ms = float(em)
                except Exception:
                    recorded_ms = None
                summary_rows.append({
                    'trial_id': trial,
                    'timer_name': timer,
                    'start_ts': '',
                    'stop_ts': ts.isoformat(),
                    'recorded_elapsed_ms': recorded_ms,
                    'computed_elapsed_ms': '',
                    'event': ev,
                    'result': res,
                    'expired': True if ev == 'timeout' else False
                })
    # remaining starts without stop
    while q:
        st = q.popleft()
        summary_rows.append({
            'trial_id': trial,
            'timer_name': timer,
            'start_ts': st['start_ts'].isoformat(),
            'stop_ts': '',
            'recorded_elapsed_ms': '',
            'computed_elapsed_ms': '',
            'event': '',
            'result': '',
            'expired': False
        })

# write summary CSV
summary_out = Path(f'timer_summary_{p.stem}.csv')
import csv as _csv
if summary_rows:
    keys = ['trial_id','timer_name','start_ts','stop_ts','recorded_elapsed_ms','computed_elapsed_ms','event','result','expired']
    with summary_out.open('w', newline='') as f:
        w = _csv.DictWriter(f, fieldnames=keys)
        w.writeheader()
        for r in summary_rows:
            w.writerow(r)

    print('\nWrote paired summary to', summary_out)
    # quick stats
    total_pairs = sum(1 for r in summary_rows if r['start_ts'] and r['stop_ts'])
    total_unmatched_stops = sum(1 for r in summary_rows if (not r['start_ts']) and r['stop_ts'])
    total_unmatched_starts = sum(1 for r in summary_rows if r['start_ts'] and (not r['stop_ts']))
    total_expired = sum(1 for r in summary_rows if r['event']=='timeout')
    print('Summary: pairs=', total_pairs, 'unmatched_stops=', total_unmatched_stops, 'unmatched_starts=', total_unmatched_starts, 'expired=', total_expired)
else:
    print('\nNo summary rows generated')
