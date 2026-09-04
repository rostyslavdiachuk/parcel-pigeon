"""Use-case functions shared by the HTTP routers and the seed script."""

from __future__ import annotations

import secrets
from datetime import UTC, datetime, timedelta

from sqlalchemy import select
from sqlalchemy.orm import Session

from app.broker import publish_event
from app.config import get_settings
from app.metrics import SHIPMENTS_CREATED, STATUS_CHANGED
from app.models import ScanEvent, Shipment
from app.shipment_status import CREATED, InvalidTransition, next_status


def generate_tracking_number() -> str:
    return "PP-" + secrets.token_hex(4).upper()


def create_shipment(
    db: Session,
    *,
    recipient_name: str,
    recipient_email: str,
    origin: str,
    destination: str,
) -> Shipment:
    now = datetime.now(UTC)
    shipment = Shipment(
        tracking_number=generate_tracking_number(),
        recipient_name=recipient_name,
        recipient_email=recipient_email,
        origin=origin,
        destination=destination,
        status=CREATED,
        eta=now + timedelta(hours=get_settings().eta_hours),
    )
    db.add(shipment)
    db.commit()
    db.refresh(shipment)

    SHIPMENTS_CREATED.inc()
    publish_event(
        "shipment.created",
        _event_payload(shipment, previous_status=None, location=shipment.origin, occurred_at=now),
    )
    return shipment


def get_shipment(db: Session, shipment_id: str) -> Shipment | None:
    return db.get(Shipment, shipment_id)


def get_by_tracking_number(db: Session, tracking_number: str) -> Shipment | None:
    return db.scalar(select(Shipment).where(Shipment.tracking_number == tracking_number))


def list_shipments(db: Session, limit: int = 100) -> list[Shipment]:
    stmt = select(Shipment).order_by(Shipment.created_at.desc()).limit(limit)
    return list(db.scalars(stmt))


def add_scan(
    db: Session,
    shipment: Shipment,
    *,
    event_type: str,
    location: str,
    note: str | None = None,
    occurred_at: datetime | None = None,
) -> ScanEvent:
    """Record a scan event, applying the status transition it implies.

    Raises :class:`app.shipment_status.InvalidTransition` on an illegal move.
    """
    previous_status = shipment.status
    new_status = next_status(previous_status, event_type)  # may raise InvalidTransition
    occurred = occurred_at or datetime.now(UTC)

    scan = ScanEvent(
        shipment_id=shipment.id,
        event_type=event_type,
        location=location,
        note=note,
        occurred_at=occurred,
    )
    db.add(scan)

    changed = new_status != previous_status
    shipment.status = new_status
    if new_status == "OUT_FOR_DELIVERY":
        shipment.eta = occurred + timedelta(hours=8)
    if new_status == "DELIVERED":
        shipment.eta = occurred
    db.commit()
    db.refresh(scan)
    db.refresh(shipment)

    if changed:
        STATUS_CHANGED.labels(new_status).inc()
    publish_event(
        "shipment.status_changed" if changed else "shipment.scan_added",
        _event_payload(
            shipment,
            previous_status=previous_status,
            location=location,
            occurred_at=occurred,
            event_type=event_type,
        ),
    )
    return scan


def _event_payload(
    shipment: Shipment,
    *,
    previous_status: str | None,
    location: str,
    occurred_at: datetime,
    event_type: str | None = None,
) -> dict:
    return {
        "trackingNumber": shipment.tracking_number,
        "status": shipment.status,
        "previousStatus": previous_status,
        "eventType": event_type,
        "location": location,
        "occurredAt": occurred_at.isoformat(),
        "recipientEmail": shipment.recipient_email,
        "recipientName": shipment.recipient_name,
        "origin": shipment.origin,
        "destination": shipment.destination,
        "eta": shipment.eta.isoformat() if shipment.eta else None,
    }


__all__ = [
    "InvalidTransition",
    "add_scan",
    "create_shipment",
    "generate_tracking_number",
    "get_by_tracking_number",
    "get_shipment",
    "list_shipments",
]
