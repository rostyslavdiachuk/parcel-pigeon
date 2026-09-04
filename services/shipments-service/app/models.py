from __future__ import annotations

import uuid
from datetime import UTC, datetime

from sqlalchemy import DateTime, ForeignKey, String, Text
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.db import Base
from app.shipment_status import CREATED


def _uuid() -> str:
    return str(uuid.uuid4())


def _now() -> datetime:
    return datetime.now(UTC)


class Shipment(Base):
    __tablename__ = "shipment"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=_uuid)
    tracking_number: Mapped[str] = mapped_column(String(20), unique=True, index=True)
    recipient_name: Mapped[str] = mapped_column(String(120))
    recipient_email: Mapped[str] = mapped_column(String(200))
    origin: Mapped[str] = mapped_column(String(120))
    destination: Mapped[str] = mapped_column(String(120))
    status: Mapped[str] = mapped_column(String(20), default=CREATED, index=True)
    eta: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=_now)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=_now, onupdate=_now
    )

    scans: Mapped[list[ScanEvent]] = relationship(
        back_populates="shipment",
        cascade="all, delete-orphan",
        order_by="ScanEvent.occurred_at",
    )


class ScanEvent(Base):
    __tablename__ = "scan_event"

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=_uuid)
    shipment_id: Mapped[str] = mapped_column(
        ForeignKey("shipment.id", ondelete="CASCADE"), index=True
    )
    event_type: Mapped[str] = mapped_column(String(40))
    location: Mapped[str] = mapped_column(String(120))
    note: Mapped[str | None] = mapped_column(Text, nullable=True)
    occurred_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=_now)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=_now)

    shipment: Mapped[Shipment] = relationship(back_populates="scans")
