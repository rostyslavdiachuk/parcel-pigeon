from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session

from app import service
from app.db import get_session
from app.schemas import (
    ScanCreate,
    ScanOut,
    ShipmentCreate,
    ShipmentOut,
    ShipmentWithScans,
)
from app.shipment_status import InvalidTransition

router = APIRouter(tags=["shipments"])


@router.post("/shipments", response_model=ShipmentOut, status_code=status.HTTP_201_CREATED)
def create_shipment(body: ShipmentCreate, db: Session = Depends(get_session)) -> ShipmentOut:
    shipment = service.create_shipment(
        db,
        recipient_name=body.recipient_name,
        recipient_email=body.recipient_email,
        origin=body.origin,
        destination=body.destination,
    )
    return shipment


@router.get("/shipments", response_model=list[ShipmentOut])
def list_shipments(db: Session = Depends(get_session)) -> list[ShipmentOut]:
    return service.list_shipments(db)


@router.get("/shipments/{shipment_id}", response_model=ShipmentWithScans)
def get_shipment(shipment_id: str, db: Session = Depends(get_session)) -> ShipmentWithScans:
    shipment = service.get_shipment(db, shipment_id)
    if shipment is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "shipment not found")
    return shipment


@router.get("/shipments/{shipment_id}/scans", response_model=list[ScanOut])
def list_scans(shipment_id: str, db: Session = Depends(get_session)) -> list[ScanOut]:
    shipment = service.get_shipment(db, shipment_id)
    if shipment is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "shipment not found")
    return shipment.scans


@router.post(
    "/shipments/{shipment_id}/scans",
    response_model=ScanOut,
    status_code=status.HTTP_201_CREATED,
)
def add_scan(
    shipment_id: str, body: ScanCreate, db: Session = Depends(get_session)
) -> ScanOut:
    shipment = service.get_shipment(db, shipment_id)
    if shipment is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "shipment not found")
    try:
        return service.add_scan(
            db,
            shipment,
            event_type=body.event_type,
            location=body.location,
            note=body.note,
            occurred_at=body.occurred_at,
        )
    except InvalidTransition as exc:
        raise HTTPException(status.HTTP_409_CONFLICT, str(exc)) from exc


@router.get("/internal/track/{tracking_number}", response_model=ShipmentWithScans)
def internal_track(tracking_number: str, db: Session = Depends(get_session)) -> ShipmentWithScans:
    """Fallback lookup used by tracking-service on a cache miss."""
    shipment = service.get_by_tracking_number(db, tracking_number)
    if shipment is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "shipment not found")
    return shipment
