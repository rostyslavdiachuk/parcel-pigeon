"""Pure shipment status-transition rules.

This module has no I/O and no framework imports on purpose: it is the single
"business rule" unit under test and the natural place to point students at when
talking about testable design.
"""

from __future__ import annotations

# Lifecycle states of a shipment.
CREATED = "CREATED"
IN_TRANSIT = "IN_TRANSIT"
OUT_FOR_DELIVERY = "OUT_FOR_DELIVERY"
DELIVERED = "DELIVERED"
EXCEPTION = "EXCEPTION"

STATUSES: tuple[str, ...] = (CREATED, IN_TRANSIT, OUT_FOR_DELIVERY, DELIVERED, EXCEPTION)

# Scan events an operator can record, mapped to the status they drive the
# shipment into.
EVENT_TO_STATUS: dict[str, str] = {
    "PICKED_UP": IN_TRANSIT,
    "IN_TRANSIT": IN_TRANSIT,
    "ARRIVED_AT_FACILITY": IN_TRANSIT,
    "OUT_FOR_DELIVERY": OUT_FOR_DELIVERY,
    "DELIVERED": DELIVERED,
    "DELIVERY_FAILED": EXCEPTION,
    "RETURNED": EXCEPTION,
}

EVENT_TYPES: tuple[str, ...] = tuple(EVENT_TO_STATUS)

# Which target states are reachable from a given state.
_ALLOWED: dict[str, set[str]] = {
    CREATED: {IN_TRANSIT, EXCEPTION},
    IN_TRANSIT: {IN_TRANSIT, OUT_FOR_DELIVERY, EXCEPTION},
    OUT_FOR_DELIVERY: {OUT_FOR_DELIVERY, DELIVERED, EXCEPTION, IN_TRANSIT},
    EXCEPTION: {IN_TRANSIT, OUT_FOR_DELIVERY, DELIVERED},
    DELIVERED: set(),  # terminal
}


class InvalidTransition(ValueError):
    """Raised when a scan event would move a shipment into an illegal state."""


def is_terminal(status: str) -> bool:
    return status == DELIVERED


def target_status_for_event(event_type: str) -> str:
    try:
        return EVENT_TO_STATUS[event_type]
    except KeyError:
        raise InvalidTransition(f"unknown event_type {event_type!r}") from None


def next_status(current: str, event_type: str) -> str:
    """Return the status a shipment should have after ``event_type``.

    Raises :class:`InvalidTransition` if the move is not allowed.
    """
    target = target_status_for_event(event_type)
    if target == current and target in _ALLOWED.get(current, set()):
        return target
    if target not in _ALLOWED.get(current, set()):
        raise InvalidTransition(
            f"cannot move from {current} to {target} (event {event_type})"
        )
    return target
