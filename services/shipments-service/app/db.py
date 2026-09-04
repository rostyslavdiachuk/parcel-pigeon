from __future__ import annotations

from collections.abc import Iterator

from sqlalchemy import create_engine
from sqlalchemy.orm import DeclarativeBase, Session, sessionmaker
from sqlalchemy.pool import StaticPool

from app.config import get_settings


class Base(DeclarativeBase):
    pass


_settings = get_settings()
_kwargs: dict = {"pool_pre_ping": True}
if _settings.database_url.startswith("sqlite"):
    # The test suite runs on SQLite; keep a single shared connection so an
    # in-memory database is visible across FastAPI's threadpool workers.
    _kwargs = {
        "connect_args": {"check_same_thread": False},
        "poolclass": StaticPool,
    }
engine = create_engine(_settings.database_url, **_kwargs)
SessionLocal = sessionmaker(bind=engine, autoflush=False, expire_on_commit=False)


def get_session() -> Iterator[Session]:
    session = SessionLocal()
    try:
        yield session
    finally:
        session.close()
