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

        if not (config.UE_HEIGHT_MIN_M <= self.height_m <= config.UE_HEIGHT_MAX_M):
            raise ValueError(
                "Aerial UE height is outside the research range "
                f"({config.UE_HEIGHT_MIN_M:g}-{config.UE_HEIGHT_MAX_M:g} m)"
            )
