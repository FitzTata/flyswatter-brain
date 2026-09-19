import numpy as np
import pytest
from worker import FRAME_SIZE, action_from_decode, render_frame


@pytest.mark.parametrize(
    ("difference_hz", "gate_spikes", "expected"),
    [
        (5.0, 1, "turn_right"),
        (-5.0, 1, "turn_left"),
        (0.5, 1, "escape"),
        (5.0, 0, "straight"),
    ],
)
def test_action_from_decode(difference_hz, gate_spikes, expected):
    decoded = {"difference_hz": difference_hz, "gate_spikes": gate_spikes}

    assert action_from_decode(decoded) == expected


def test_render_frame():
    observation = {
        "fly": {"position": {"x": 0.5, "y": 0.5}, "radius": 0.025},
        "swatter": {
            "position": {"x": 0.75, "y": 0.25},
            "radius": 0.09,
            "attacking": True,
        },
    }

    frame = render_frame(observation)

    assert frame.shape == (FRAME_SIZE[1], FRAME_SIZE[0], 3)
    assert frame.dtype == np.uint8
