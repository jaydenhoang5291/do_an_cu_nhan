# Cellular Network Simulation

This project is a Python-based cellular network simulator that visualizes and logs the behavior of User Equipments (UEs) moving within a Manhattan grid coverage area. It models ground base station (BS) connectivity, radio signal propagation, path loss, shadow fading, and handing over between cells. UE height can be configured to study ground or aerial UE links.

## Project Structure

- `main.py`: The entry point of the application. Prompts the user for simulation parameters (area size, number of UEs, grid spacing, UE height, link-line display) and launches the simulation.
- `simulation.py`: The core `CellularNetworkReceivedPower` simulation class. Orchestrates the animation, UI controls (Start/Stop/Restart), applies mobility and radio models to each UE, and tracks the state of the network at each step.
- `config.py`: Contains global configuration variables such as radio parameters (transmit power `PTX`, gain), path loss exponents, shadow fading standard deviations, UE stop-and-go duration settings, and noise power.
- `mobility.py`: Implements the `GridMobility` class for the Manhattan mobility model. Keeps UEs constrained to the generated road grid, handles intersections, directional turns, and simulating realistic stop-and-go behavior after turns.
- `radio_models.py`: Implements the `RadioModel` class. Handles distance calculation, line-of-sight (LOS) and non-line-of-sight (NLOS) path loss, correlated shadow fading penalty, received power, Signal-to-Interference-Plus-Noise Ratio (SINR), and base station association (incorporating Handover Margin - HOM).
- `hexagons.py`: Generates a hexagonal grid topology for deploying Ground Base Stations across the simulation map (`build_hex_cover`).
- `logger.py`: Implements `SimulationLogger` to continuously track information (position, direction, connected BS, received power, SINR, speed, handover events) per UE at each simulation step, saving the result sequentially into a `.csv` file in the `data/` directory.
- `utils.py`: Helper functions for reading typed user inputs (`_read_float`, `_read_int`).

## Features

- **Manhattan Mobility Model**: UEs move along horizontal and vertical grid lines, making realistic decisions at intersections.
- **Stop-and-Go Mechanism**: UEs dynamically adjust their speeds, ramping down and pausing after a configurable number of turns to simulate real-world urban traffic or pedestrian delays.
- **Hexagonal Ground BS Grid**: Ground Base Stations are systematically plotted using an axial-to-pixel hexagonal generation algorithm.
- **Aerial UE Option**: Allows users to simulate elevated UEs with UMa-AV LOS probability, path loss, and shadow fading models.
- **Radio Propagation Engine**: Accounts for frequency-specific Free Space Path Loss (FSPL), Shadow Fading with geographical correlation and turn penalties, interference from neighboring cells, and SINR.
- **Handover Management**: Evaluates the received signal strength to perform cell handovers when a candidate BS signal strength exceeds the current BS by the Handover Margin (`HOM`).
- **Interactive Visualization**: Uses `matplotlib` to render a live playback of the simulation showing paths, UE movement, BS coverage areas, and active link connections.
- **Automated Data Logging**: Saves trace parameters per frame into a unified CSV format for post-simulation analysis.

## Usage

Run the main file to start the simulation:

```bash
python main.py
```

You will be prompted to enter parameters:
1. Rectangle Length (m) and Width (m) for the simulation area.
2. Number of UEs to simulate.
3. Road grid spacing (m) - dictates the Manhattan layout.
4. Use aerial UE height (y/N) - if yes, prompts for UE height.
5. Show UE - BS connection lines (y/N).

Once the inputs are collected, a matplotlib window will load. You can **Stop/Continue** or **Restart** the simulation using the provided UI buttons. When you close the window (or upon simulation end), the logged data is saved to a timestamped CSV inside the `data/` folder.

## Requirements
- Python 3.x
- `numpy`
- `pandas`
- `matplotlib`
