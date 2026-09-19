from __future__ import annotations

import hashlib
import json
import os
import sys
import time
from pathlib import Path
from typing import Any

import numpy as np
from PIL import Image, ImageDraw

FRAME_SIZE = (320, 180)
THRESHOLD_HZ = 2.0


def main() -> None:
    source = Path(os.environ["FLYSWATTER_STONKFLY_SOURCE"]).resolve()
    sys.path.insert(0, str(source))

    from stonkfly.neural.common import annotations
    from stonkfly.neural.controller import Decoder
    from stonkfly.neural.visual import VisualMemoryBrain

    brain = VisualMemoryBrain()
    brain.weights_frozen = True
    decoder = Decoder(brain.ids, annotations(brain.ids), THRESHOLD_HZ)
    send(
        {
            "type": "ready",
            "model": "MaleCNS v1.0",
            "neurons": int(brain.n),
            "connections": int(len(brain.post)),
        }
    )

    for line in sys.stdin:
        try:
            request = json.loads(line)
            if request.get("type") != "step":
                raise ValueError("unsupported request type")
            duration_ms = float(request.get("duration_ms", 50))
            frame = render_frame(request["observation"])
            started = time.perf_counter()
            counts, kernel_seconds = brain.rgb_step(frame, duration_ms, learning=False)
            decoded = decoder.decode(counts, duration_ms / 1000)
            send(
                {
                    "type": "decision",
                    "action": action_from_decode(decoded),
                    "neural_activity": activity(
                        brain,
                        decoder,
                        counts,
                        duration_ms,
                        decoded,
                        kernel_seconds,
                        time.perf_counter() - started,
                        frame,
                    ),
                }
            )
        except Exception as error:
            send({"type": "error", "error": str(error)})


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
    draw.line((sx + radius * 0.7, sy + radius * 0.7, sx + radius * 2, sy + radius * 2), fill=color, width=line_width)

    return np.asarray(image, dtype=np.uint8)


def action_from_decode(decoded: dict[str, Any]) -> str:
    if decoded["gate_spikes"] == 0:
        return "straight"
    if abs(decoded["difference_hz"]) < THRESHOLD_HZ:
        return "escape"
    return "turn_right" if decoded["difference_hz"] > 0 else "turn_left"


def activity(
    brain: Any,
    decoder: Any,
    counts: np.ndarray,
    duration_ms: float,
    decoded: dict[str, Any],
    kernel_seconds: float,
    total_seconds: float,
    frame: np.ndarray,
) -> dict[str, Any]:
    seconds = duration_ms / 1000
    nodes = []
    for label, indices in (
        ("DNp20-L", decoder.left),
        ("DNp20-R", decoder.right),
        ("DNpe017", decoder.gate),
    ):
        for index in indices:
            spikes = int(counts[index])
            nodes.append(
                {
                    "id": str(brain.ids[index]),
                    "label": label,
                    "spikes": spikes,
                    "rate_hz": spikes / seconds,
                }
            )
    return {
        "model": "MaleCNS v1.0",
        "model_time_ms": float(brain.sim_ms),
        "left_hz": decoded["left_hz"],
        "right_hz": decoded["right_hz"],
        "difference_hz": decoded["difference_hz"],
        "gate_spikes": decoded["gate_spikes"],
        "total_spikes": int(counts.sum()),
        "kernel_seconds": kernel_seconds,
        "step_seconds": total_seconds,
        "input_sha256": hashlib.sha256(frame.tobytes()).hexdigest(),
        "nodes": nodes,
    }


def point(position: dict[str, Any], width: int, height: int) -> tuple[int, int]:
    x = min(1.0, max(0.0, float(position["x"])))
    y = min(1.0, max(0.0, float(position["y"])))
    return round(x * (width - 1)), round(y * (height - 1))


def send(message: dict[str, Any]) -> None:
    print(json.dumps(message, separators=(",", ":"), allow_nan=False), flush=True)


if __name__ == "__main__":
    main()
