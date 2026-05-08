"""Aerial UE metadata model used by the simulator."""

from dataclasses import dataclass

import config


@dataclass
class AerialUE:
    height_m: float
    speed_mps: float = 0.0
    direction_rad: float = 0.0
    ue_type: str = "aerial"

    def __post_init__(self):
        self.height_m = float(self.height_m)
        self.speed_mps = float(self.speed_mps)
        self.direction_rad = float(self.direction_rad)

        if self.height_m not in config.AERIAL_UE_HEIGHTS_M:
            allowed = ", ".join(f"{h:g}" for h in config.AERIAL_UE_HEIGHTS_M)
            raise ValueError(f"Aerial UE height must be one of: {allowed} m")
