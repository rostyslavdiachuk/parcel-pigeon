from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel, ConfigDict, EmailStr, Field

from app.shipment_status import EVENT_TYPES


class ShipmentCreate(BaseModel):
    recipient_name: str = Field(min_length=1, max_length=120)
    recipient_email: EmailStr
    origin: str = Field(min_length=1, max_length=120)
    destination: str = Field(min_length=1, max_length=120)


class ScanCreate(BaseModel):
    event_type: str = Field(examples=list(EVENT_TYPES))
    location: str = Field(min_length=1, max_length=120)
    note: str | None = Field(default=None, max_length=2000)
    occurred_at: datetime | None = None


class ScanOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    event_type: str
    location: str
    note: str | None
    occurred_at: datetime


class ShipmentOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    tracking_number: str
    recipient_name: str
    recipient_email: str
    origin: str
    destination: str
    status: str
    eta: datetime | None
    created_at: datetime
    updated_at: datetime


class ShipmentWithScans(ShipmentOut):
    scans: list[ScanOut] = []
