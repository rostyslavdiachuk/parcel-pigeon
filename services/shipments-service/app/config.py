from __future__ import annotations

from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_prefix="SHIPMENTS_", env_file=".env", extra="ignore")

    database_url: str = "postgresql+psycopg://parcelpigeon:parcelpigeon@postgres:5432/shipments"
    rabbitmq_url: str = "amqp://guest:guest@rabbitmq:5672/"
    rabbitmq_exchange: str = "parcelpigeon"
    # When false (tests), publishing is skipped instead of attempting a broker connection.
    publish_events: bool = True
    eta_hours: int = 72
    log_level: str = "INFO"


@lru_cache
def get_settings() -> Settings:
    return Settings()
