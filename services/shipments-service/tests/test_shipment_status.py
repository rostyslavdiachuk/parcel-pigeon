from __future__ import annotations

import pytest

from app.shipment_status import (
    CREATED,
    DELIVERED,
    EXCEPTION,
    IN_TRANSIT,
    OUT_FOR_DELIVERY,
    InvalidTransition,
    is_terminal,
    next_status,
)


@pytest.mark.parametrize(
    ("current", "event", "expected"),
    [
        (CREATED, "PICKED_UP", IN_TRANSIT),
        (IN_TRANSIT, "ARRIVED_AT_FACILITY", IN_TRANSIT),
        (IN_TRANSIT, "OUT_FOR_DELIVERY", OUT_FOR_DELIVERY),
        (OUT_FOR_DELIVERY, "DELIVERED", DELIVERED),
        (OUT_FOR_DELIVERY, "DELIVERY_FAILED", EXCEPTION),
        (EXCEPTION, "OUT_FOR_DELIVERY", OUT_FOR_DELIVERY),
    ],
)
def test_valid_transitions(current: str, event: str, expected: str) -> None:
    assert next_status(current, event) == expected


@pytest.mark.parametrize(
    ("current", "event"),
    [
        (CREATED, "DELIVERED"),          # can't deliver something never in transit
        (CREATED, "OUT_FOR_DELIVERY"),
        (DELIVERED, "PICKED_UP"),        # terminal
        (DELIVERED, "IN_TRANSIT"),
        (IN_TRANSIT, "DELIVERED"),       # must go out for delivery first
    ],
)
def test_illegal_transitions_raise(current: str, event: str) -> None:
    with pytest.raises(InvalidTransition):
        next_status(current, event)


def test_unknown_event_type() -> None:
    with pytest.raises(InvalidTransition):
        next_status(IN_TRANSIT, "TELEPORTED")


def test_is_terminal() -> None:
    assert is_terminal(DELIVERED)
    assert not is_terminal(IN_TRANSIT)
