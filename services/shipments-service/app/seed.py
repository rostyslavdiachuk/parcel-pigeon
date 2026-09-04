"""Idempotent demo data loader.

Run after migrations:  ``python -m app.seed``  (add ``--force`` to wipe first).
Uses stable tracking numbers so docs and the smoke test can refer to them.
"""

from __future__ import annotations

import sys
from datetime import UTC, datetime, timedelta

from app.db import SessionLocal
from app.models import Shipment
from app.service import add_scan

DEMO = [
    {
        "tracking_number": "PP-DEMO0001",
        "recipient_name": "Ada Lovelace",
        "recipient_email": "ada@example.com",
        "origin": "London",
        "destination": "Paris",
        "scans": [
            ("PICKED_UP", "London Hub", -48),
            ("ARRIVED_AT_FACILITY", "Calais Sort Center", -30),
            ("OUT_FOR_DELIVERY", "Paris Depot", -6),
            ("DELIVERED", "Paris, 5th arr.", -2),
        ],
    },
    {
        "tracking_number": "PP-DEMO0002",
        "recipient_name": "Alan Turing",
        "recipient_email": "alan@example.com",
        "origin": "Manchester",
        "destination": "Berlin",
        "scans": [
            ("PICKED_UP", "Manchester Hub", -20),
            ("IN_TRANSIT", "Rotterdam", -8),
        ],
    },
    {
        "tracking_number": "PP-DEMO0003",
        "recipient_name": "Grace Hopper",
        "recipient_email": "grace@example.com",
        "origin": "New York",
        "destination": "Boston",
        "scans": [
            ("PICKED_UP", "NYC Hub", -12),
            ("OUT_FOR_DELIVERY", "Boston Depot", -3),
            ("DELIVERY_FAILED", "Boston, recipient absent", -1),
        ],
    },
    {
        "tracking_number": "PP-DEMO0004",
        "recipient_name": "Katherine Johnson",
        "recipient_email": "katherine@example.com",
        "origin": "Hampton",
        "destination": "Houston",
        "scans": [],
    },
]


def run(force: bool = False) -> None:
    db = SessionLocal()
    try:
        existing = {s.tracking_number for s in db.query(Shipment).all()}
        if existing and not force:
            print(f"seed: {len(existing)} shipments already present, skipping (use --force)")
            return
        if force:
            db.query(Shipment).delete()
            db.commit()
            existing = set()

        now = datetime.now(UTC)
        for spec in DEMO:
            if spec["tracking_number"] in existing:
                continue
            shipment = Shipment(
                tracking_number=spec["tracking_number"],
                recipient_name=spec["recipient_name"],
                recipient_email=spec["recipient_email"],
                origin=spec["origin"],
                destination=spec["destination"],
                eta=now + timedelta(hours=72),
            )
            db.add(shipment)
            db.commit()
            db.refresh(shipment)
            for event_type, location, hours in spec["scans"]:
                add_scan(
                    db,
                    shipment,
                    event_type=event_type,
                    location=location,
                    occurred_at=now + timedelta(hours=hours),
                )
            print(f"seed: {shipment.tracking_number} -> {shipment.status}")
    finally:
        db.close()


if __name__ == "__main__":
    run(force="--force" in sys.argv)
