def reinforcement_circuit_key(reinforcement: str) -> str | None:
    if reinforcement == "reward":
        return "reward"
    if reinforcement == "aversive":
        return "aversive"
    if reinforcement in ("", "none"):
        return None
    raise ValueError(f"unknown reinforcement: {reinforcement}")
