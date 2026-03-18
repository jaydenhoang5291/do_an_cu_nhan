import math
from dataclasses import dataclass


def _read_float(prompt: str, default: float) -> float:
    try:
        s = input(prompt).strip()
        return float(s) if s else float(default)
    except Exception:
        return float(default)

def _read_int(prompt: str, default: int) -> int:
    try:
        s = input(prompt).strip()
        return int(s) if s else int(default)
    except Exception:
        return int(default)
