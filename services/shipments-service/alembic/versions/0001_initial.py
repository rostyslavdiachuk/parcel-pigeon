"""initial schema

Revision ID: 0001
Revises:
Create Date: 2026-09-01
"""
from __future__ import annotations

from collections.abc import Sequence

import sqlalchemy as sa

from alembic import op

revision: str = "0001"
down_revision: str | None = None
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "shipment",
        sa.Column("id", sa.String(length=36), primary_key=True),
        sa.Column("tracking_number", sa.String(length=20), nullable=False),
        sa.Column("recipient_name", sa.String(length=120), nullable=False),
        sa.Column("recipient_email", sa.String(length=200), nullable=False),
        sa.Column("origin", sa.String(length=120), nullable=False),
        sa.Column("destination", sa.String(length=120), nullable=False),
        sa.Column("status", sa.String(length=20), nullable=False),
        sa.Column("eta", sa.DateTime(timezone=True), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False),
    )
    op.create_index("ix_shipment_tracking_number", "shipment", ["tracking_number"], unique=True)
    op.create_index("ix_shipment_status", "shipment", ["status"])

    op.create_table(
        "scan_event",
        sa.Column("id", sa.String(length=36), primary_key=True),
        sa.Column("shipment_id", sa.String(length=36), nullable=False),
        sa.Column("event_type", sa.String(length=40), nullable=False),
        sa.Column("location", sa.String(length=120), nullable=False),
        sa.Column("note", sa.Text(), nullable=True),
        sa.Column("occurred_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(["shipment_id"], ["shipment.id"], ondelete="CASCADE"),
    )
    op.create_index("ix_scan_event_shipment_id", "scan_event", ["shipment_id"])


def downgrade() -> None:
    op.drop_table("scan_event")
    op.drop_table("shipment")
