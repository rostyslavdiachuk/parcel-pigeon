"""RabbitMQ publisher for shipment domain events.

Kept deliberately small: a lazily-opened blocking connection with a
reconnect-on-failure publish. In tests ``publish_events`` is false and
:func:`publish_event` becomes a no-op that still records the call for assertions.
"""

from __future__ import annotations

import json
import logging
import threading
from typing import Any

import pika

from app.config import get_settings
from app.metrics import EVENTS_PUBLISHED

log = logging.getLogger("shipments.broker")

_lock = threading.Lock()
_connection: pika.BlockingConnection | None = None
_channel: Any = None

# Test hook: every publish attempt is appended here regardless of transport.
published: list[tuple[str, dict]] = []


def _ensure_channel() -> Any:
    global _connection, _channel
    settings = get_settings()
    if _channel is not None and _channel.is_open:
        return _channel
    _connection = pika.BlockingConnection(pika.URLParameters(settings.rabbitmq_url))
    _channel = _connection.channel()
    _channel.exchange_declare(
        exchange=settings.rabbitmq_exchange, exchange_type="topic", durable=True
    )
    return _channel


def publish_event(routing_key: str, payload: dict) -> None:
    settings = get_settings()
    published.append((routing_key, payload))
    if not settings.publish_events:
        return

    body = json.dumps(payload, default=str).encode()
    with _lock:
        for attempt in (1, 2):
            try:
                channel = _ensure_channel()
                channel.basic_publish(
                    exchange=settings.rabbitmq_exchange,
                    routing_key=routing_key,
                    body=body,
                    properties=pika.BasicProperties(
                        content_type="application/json", delivery_mode=2
                    ),
                )
                EVENTS_PUBLISHED.labels(routing_key, "ok").inc()
                return
            except Exception as exc:  # noqa: BLE001 - broker errors are broad
                global _connection, _channel
                _connection = _channel = None
                log.warning("publish failed (attempt %s): %s", attempt, exc)
        EVENTS_PUBLISHED.labels(routing_key, "error").inc()


def check_broker() -> bool:
    try:
        _ensure_channel()
        return True
    except Exception:  # noqa: BLE001
        return False


def close() -> None:
    global _connection, _channel
    try:
        if _connection is not None and _connection.is_open:
            _connection.close()
    finally:
        _connection = _channel = None
