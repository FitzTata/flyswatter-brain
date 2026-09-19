from typing import Any

import numpy as np
from PIL import Image, ImageDraw

FRAME_SIZE = (320, 180)
THRESHOLD_HZ = 2.0


def render_frame(observation: dict[str, Any]) -> np.ndarray:
    image = Image.new("RGB", FRAME_SIZE, (226, 232, 219))
    draw = ImageDraw.Draw(image)
    width, height = FRAME_SIZE

    for x in range(0, width, 24):
        draw.line((x, 0, x, height), fill=(207, 216, 200), width=1)
    for y in range(0, height, 24):
        draw.line((0, y, width, y), fill=(207, 216, 200), width=1)

    fly = observation["fly"]
    fx, fy = point(fly["position"], width, height)
    fly_radius = max(3, round(float(fly["radius"]) * min(FRAME_SIZE)))
    draw.ellipse(
        (fx - fly_radius, fy - fly_radius, fx + fly_radius, fy + fly_radius),
        fill=(36, 46, 30),
        outline=(92, 118, 68),
        width=1,
    )

    swatter = observation["swatter"]
    sx, sy = point(swatter["position"], width, height)
    radius = max(6, round(float(swatter["radius"]) * min(FRAME_SIZE)))
    color = (208, 48, 30) if swatter["attacking"] else (238, 103, 55)
    line_width = 4 if swatter["attacking"] else 2
    draw.ellipse(
        (sx - radius, sy - radius, sx + radius, sy + radius),
        outline=color,
        width=line_width,
    )
    draw.line(
        (sx + radius * 0.7, sy + radius * 0.7, sx + radius * 2, sy + radius * 2),
        fill=color,
        width=line_width,
    )

    return np.asarray(image, dtype=np.uint8)


def action_from_decode(decoded: dict[str, Any]) -> str:
    if decoded["gate_spikes"] == 0:
        return "straight"
    if abs(decoded["difference_hz"]) < THRESHOLD_HZ:
        return "escape"
    return "turn_right" if decoded["difference_hz"] > 0 else "turn_left"


def point(position: dict[str, Any], width: int, height: int) -> tuple[int, int]:
    x = min(1.0, max(0.0, float(position["x"])))
    y = min(1.0, max(0.0, float(position["y"])))
    return round(x * (width - 1)), round(y * (height - 1))
