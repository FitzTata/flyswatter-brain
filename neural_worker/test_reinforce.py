import pytest
from reinforce import reinforcement_circuit_key


@pytest.mark.parametrize(
    ("value", "expected"),
    [
        ("reward", "reward"),
        ("aversive", "aversive"),
        ("none", None),
        ("", None),
    ],
)
def test_reinforcement_circuit_key(value, expected):
    assert reinforcement_circuit_key(value) == expected


def test_reinforcement_circuit_key_rejects_unknown():
    with pytest.raises(ValueError):
        reinforcement_circuit_key("pain")
