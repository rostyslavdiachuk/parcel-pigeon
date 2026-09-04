from __future__ import annotations

import os

os.environ.setdefault("SHIPMENTS_DATABASE_URL", "sqlite+pysqlite:///:memory:")
os.environ.setdefault("SHIPMENTS_PUBLISH_EVENTS", "false")

import pytest  # noqa: E402
from fastapi.testclient import TestClient  # noqa: E402

from app import broker  # noqa: E402
from app.db import Base, engine  # noqa: E402
from app.main import app  # noqa: E402


@pytest.fixture(autouse=True)
def _schema():
    Base.metadata.create_all(engine)
    broker.published.clear()
    yield
    Base.metadata.drop_all(engine)


@pytest.fixture
def client() -> TestClient:
    return TestClient(app)


@pytest.fixture
def published() -> list[tuple[str, dict]]:
    return broker.published
