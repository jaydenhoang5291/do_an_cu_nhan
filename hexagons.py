import math
from dataclasses import dataclass

@dataclass
class Hex:
    q: int
    r: int
    s: int
    def __post_init__(self):
        assert self.q + self.r + self.s == 0

def axial_to_pixel(q: int, r: int, size: float = 500.0) -> tuple[float, float]:
    x = size * (3 / 2 * q)
    y = size * (math.sqrt(3) * r + (math.sqrt(3) / 2) * q)
    return x, y

def build_hex_cover(sim_scale_factor: float, sim_width: float, sim_height: float, sim_center: float, sim_rect_xmin: float, sim_rect_xmax: float, sim_rect_ymin: float, sim_rect_ymax: float) -> list[Hex]:
    """
    Builds a hexagonal grid cover for the simulation area.
    """
    size = sim_scale_factor
    w, h = sim_width, sim_height
    q_range = range(int(-w / (1.5 * size)) - 1, int(w / (1.5 * size)) + 2)
    r_range = range(int(-h / (size * math.sqrt(3))) - 1, int(h / (size * math.sqrt(3))) + 2)
    
    hexes = []
    for q in q_range:
        for r in r_range:
            s = -q - r
            cx, cy = axial_to_pixel(q, r, size)
            cx += sim_center
            cy += sim_center
            buffer = size * math.sqrt(3) / 2
            
            # Check if hexagon center is within the simulation boundary (+ buffer)
            if (sim_rect_xmin - buffer <= cx <= sim_rect_xmax + buffer and
                sim_rect_ymin - buffer <= cy <= sim_rect_ymax + buffer):
                hexes.append(Hex(q, r, s))
                
    return hexes
