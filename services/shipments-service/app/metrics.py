from __future__ import annotations

import time

from prometheus_client import Counter, Histogram
from starlette.types import ASGIApp, Message, Receive, Scope, Send

REQUESTS = Counter(
    "http_requests_total",
    "HTTP requests handled",
    ["method", "path", "status"],
)
LATENCY = Histogram(
    "http_request_duration_seconds",
    "HTTP request latency",
    ["method", "path"],
)
SHIPMENTS_CREATED = Counter("shipments_created_total", "Shipments created")
STATUS_CHANGED = Counter(
    "shipment_status_changed_total", "Shipment status changes", ["status"]
)
EVENTS_PUBLISHED = Counter(
    "shipment_events_published_total", "Domain events published", ["routing_key", "outcome"]
)


class MetricsMiddleware:
    """Tiny ASGI middleware recording RED metrics keyed by route template."""

    def __init__(self, app: ASGIApp) -> None:
        self.app = app

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        method = scope["method"]
        start = time.perf_counter()
        status_holder = {"code": 500}

        async def _send(message: Message) -> None:
            if message["type"] == "http.response.start":
                status_holder["code"] = message["status"]
            await send(message)

        try:
            await self.app(scope, receive, _send)
        finally:
            route = scope.get("route")
            path = getattr(route, "path", scope["path"])
            LATENCY.labels(method, path).observe(time.perf_counter() - start)
            REQUESTS.labels(method, path, str(status_holder["code"])).inc()
