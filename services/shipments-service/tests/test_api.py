from __future__ import annotations


def _create(client, **overrides):
    body = {
        "recipient_name": "Test User",
        "recipient_email": "test@example.com",
        "origin": "London",
        "destination": "Paris",
    }
    body.update(overrides)
    resp = client.post("/shipments", json=body)
    assert resp.status_code == 201, resp.text
    return resp.json()


def test_create_shipment_emits_event(client, published):
    data = _create(client)
    assert data["status"] == "CREATED"
    assert data["tracking_number"].startswith("PP-")
    assert data["eta"] is not None
    assert published[-1][0] == "shipment.created"
    assert published[-1][1]["trackingNumber"] == data["tracking_number"]


def test_scan_flow_updates_status_and_publishes(client, published):
    shipment = _create(client)
    sid = shipment["id"]

    for event_type in ("PICKED_UP", "OUT_FOR_DELIVERY", "DELIVERED"):
        resp = client.post(
            f"/shipments/{sid}/scans",
            json={"event_type": event_type, "location": "Somewhere"},
        )
        assert resp.status_code == 201, resp.text

    got = client.get(f"/shipments/{sid}").json()
    assert got["status"] == "DELIVERED"
    assert [s["event_type"] for s in got["scans"]] == [
        "PICKED_UP",
        "OUT_FOR_DELIVERY",
        "DELIVERED",
    ]
    routing_keys = [rk for rk, _ in published]
    assert routing_keys.count("shipment.status_changed") == 3


def test_illegal_scan_returns_409(client):
    shipment = _create(client)
    resp = client.post(
        f"/shipments/{shipment['id']}/scans",
        json={"event_type": "DELIVERED", "location": "Nowhere"},
    )
    assert resp.status_code == 409


def test_internal_track_lookup(client):
    shipment = _create(client)
    resp = client.get(f"/internal/track/{shipment['tracking_number']}")
    assert resp.status_code == 200
    assert resp.json()["id"] == shipment["id"]

    assert client.get("/internal/track/PP-UNKNOWN").status_code == 404


def test_list_shipments(client):
    _create(client)
    _create(client)
    resp = client.get("/shipments")
    assert resp.status_code == 200
    assert len(resp.json()) == 2


def test_healthz(client):
    assert client.get("/healthz").json() == {"status": "ok"}


def test_metrics_exposed(client):
    _create(client)
    body = client.get("/metrics").text
    assert "shipments_created_total" in body
