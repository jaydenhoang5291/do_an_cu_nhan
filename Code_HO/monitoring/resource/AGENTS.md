<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Resource Monitoring Scripts

## Purpose
Python-based system resource monitoring tools for tracking CPU and RAM usage during StormSIM benchmarking sessions. Used to correlate system resource consumption with UE/gNB scaling tests against different 5G Core implementations (Open5GS, Free5GC).

## Key Files
| File | Description |
|------|-------------|
| `cpu_ram.py` | CPU/RAM usage logger using psutil - captures per-core CPU and memory metrics |
| `chart.ipynb` | Jupyter notebook for visualizing resource usage data from log files |

## Subdirectories
None

## For AI Agents

### cpu_ram.py Usage
```bash
# Run from project root or monitoring/resource directory
cd monitoring/resource
python cpu_ram.py

# Output format: logs/data-{YYYY-MM-DD_HH-MM-SS}.log
# Log format: HH:MM:SS - cpu1, cpu2, ..., cpuN, ram_percent
# Example: 22:58:48 - 12.5, 8.3, 15.2, 10.1, 45.6
```

**Log Format Details:**
- Time: `HH:MM:SS` format
- CPU values: Per-core percentage (comma-separated)
- RAM value: Last value in the comma-separated list
- Sampling interval: 1 second (hardcoded in `psutil.cpu_percent(interval=1)`)

### chart.ipynb Usage
The notebook provides `print_chart(filename, is_open5gs)` function:

```python
# Visualize Open5GS benchmark data
print_chart("../../logs/open5gs.log", True)

# Visualize Free5GC benchmark data  
print_chart("../../logs/free5gc.log", False)
```

**Visualization Features:**
- CPU average across all cores (green line)
- RAM usage (red dashed line)
- Time axis standardized to seconds from start
- Statistics: total data points, time range, task execution metrics
- Configurable: `max_line` parameter limits data points (default 2000)

**Data Processing Notes:**
- Open5GS mode applies `RAM + log(index)` adjustment for visualization
- RAM values are doubled (`*2`) in the processing
- Task execution threshold: any CPU core > 10%

### Expected Log Files
The scripts expect log files in `logs/` directory relative to execution context:
- `logs/open5gs.log` - Open5GS benchmark resource data
- `logs/free5gc.log` - Free5GC benchmark resource data

### Adding New Metrics
To extend `cpu_ram.py` with additional metrics:
```python
# Add to the monitoring loop in main()
disk = psutil.disk_usage('/').percent          # Disk usage
net_io = psutil.net_io_counters()              # Network I/O
load_avg = psutil.getloadavg()                 # Load average (Unix only)
```

## Dependencies

### Python Requirements
```
psutil>=5.9.0        # System resource monitoring
pandas>=1.5.0        # Data manipulation (notebook only)
matplotlib>=3.6.0    # Plotting (notebook only)
numpy>=1.24.0        # Numerical operations (notebook only)
```

### Installation
```bash
# Create virtual environment (recommended)
python -m venv .venv
source .venv/bin/activate  # Linux/macOS

# Install dependencies
pip install psutil pandas matplotlib numpy

# Or install from requirements if available
pip install -r requirements.txt
```

### Runtime Requirements
- Python 3.11+ (as indicated by notebook kernel)
- Access to `/proc` filesystem (Linux) for psutil
- Write permissions for `logs/` directory

<!-- MANUAL: -->
