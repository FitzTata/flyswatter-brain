import hashlib
import json
import logging
import os
import sys
import time
from collections import deque
from pathlib import Path
from typing import Any

import numpy as np

_SOURCE = Path(os.environ.get("FLYSWATTER_STONKFLY_SOURCE", "")).resolve()
if _SOURCE.is_dir():
    sys.path.insert(0, str(_SOURCE))

from stonkfly.neural.common import annotations
from stonkfly.neural.controller import Decoder
from stonkfly.neural.visual import VisualMemoryBrain

from frame import THRESHOLD_HZ, action_from_decode, render_frame

READOUT_WINDOW_MS = 100

logging.basicConfig(
    level=os.environ.get("FLYSWATTER_LOG_LEVEL", "INFO").upper(),
    format="%(levelname)s component=neural-worker %(message)s",
    stream=sys.stderr,
)
logger = logging.getLogger("neural-worker")


def main() -> None:
    brain = VisualMemoryBrain()
    brain.weights_frozen = True
    decoder = Decoder(brain.ids, annotations(brain.ids), THRESHOLD_HZ)
    history: deque[np.ndarray] = deque()
    rolling_counts = np.zeros(brain.n, dtype=np.int64)
    send(
        {
            "type": "ready",
            "model": "MaleCNS v1.0",
            "neurons": int(brain.n),
            "connections": int(len(brain.post)),
        }
    )
    logger.info("ready neurons=%s connections=%s", brain.n, len(brain.post))

    for line in sys.stdin:
        try:
            request = json.loads(line)
            if request.get("type") != "step":
                raise ValueError("unsupported request type")
            duration_ms = float(request.get("duration_ms", 50))
            frame = render_frame(request["observation"])
            started = time.perf_counter()
            counts, kernel_seconds = brain.rgb_step(frame, duration_ms, learning=False)
            history.append(counts)
            rolling_counts += counts
            max_samples = max(1, round(READOUT_WINDOW_MS / duration_ms))
            while len(history) > max_samples:
                rolling_counts -= history.popleft()
            window_ms = duration_ms * len(history)
            decoded = decoder.decode(rolling_counts, window_ms / 1000)
            send(
                {
                    "type": "decision",
                    "action": action_from_decode(decoded),
                    "neural_activity": activity(
                        brain,
                        decoder,
                        rolling_counts,
                        window_ms,
                        decoded,
                        kernel_seconds,
                        time.perf_counter() - started,
                        frame,
                    ),
                }
            )
        except Exception as error:
            logger.exception("step failed")
            send({"type": "error", "error": str(error)})


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


def send(message: dict[str, Any]) -> None:
    print(json.dumps(message, separators=(",", ":"), allow_nan=False), flush=True)


if __name__ == "__main__":
    main()
