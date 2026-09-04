import { useCallback, useEffect, useState } from 'react';
import { api } from '../api';
import type { Shipment } from '../types';
import { EVENT_TYPES } from '../types';
import { StatusBadge } from '../components/StatusBadge';

const EMPTY = { recipient_name: '', recipient_email: '', origin: '', destination: '' };

export function OpsPage() {
  const [shipments, setShipments] = useState<Shipment[]>([]);
  const [form, setForm] = useState(EMPTY);
  const [scanFor, setScanFor] = useState<string>('');
  const [scan, setScan] = useState({ event_type: EVENT_TYPES[0], location: '' });
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      setShipments(await api.listShipments());
    } catch {
      setError('Could not load shipments — is the gateway up?');
    }
  }, []);

  useEffect(() => {
    void refresh();
    const t = setInterval(refresh, 5000);
    return () => clearInterval(t);
  }, [refresh]);

  async function createShipment(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await api.createShipment(form);
      setForm(EMPTY);
      await refresh();
    } catch {
      setError('Create failed — check the form values.');
    }
  }

  async function addScan(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await api.addScan(scanFor, scan);
      setScanFor('');
      setScan({ event_type: EVENT_TYPES[0], location: '' });
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Scan failed.');
    }
  }

  return (
    <section>
      <h2>Operations dashboard</h2>
      {error && (
        <p role="alert" className="error">
          {error}
        </p>
      )}

      <form onSubmit={createShipment} className="card grid2">
        <h3 className="span2">New shipment</h3>
        <input
          aria-label="recipient name"
          placeholder="Recipient name"
          value={form.recipient_name}
          onChange={(e) => setForm({ ...form, recipient_name: e.target.value })}
          required
        />
        <input
          aria-label="recipient email"
          placeholder="Recipient email"
          type="email"
          value={form.recipient_email}
          onChange={(e) => setForm({ ...form, recipient_email: e.target.value })}
          required
        />
        <input
          aria-label="origin"
          placeholder="Origin"
          value={form.origin}
          onChange={(e) => setForm({ ...form, origin: e.target.value })}
          required
        />
        <input
          aria-label="destination"
          placeholder="Destination"
          value={form.destination}
          onChange={(e) => setForm({ ...form, destination: e.target.value })}
          required
        />
        <button type="submit" className="span2">
          Create shipment
        </button>
      </form>

      <table className="table">
        <thead>
          <tr>
            <th>Tracking #</th>
            <th>Recipient</th>
            <th>Route</th>
            <th>Status</th>
            <th>Scan</th>
          </tr>
        </thead>
        <tbody>
          {shipments.map((s) => (
            <tr key={s.id}>
              <td>
                <code>{s.tracking_number}</code>
              </td>
              <td>{s.recipient_name}</td>
              <td className="muted">
                {s.origin} → {s.destination}
              </td>
              <td>
                <StatusBadge status={s.status} />
              </td>
              <td>
                <button type="button" onClick={() => setScanFor(s.id)}>
                  Add scan
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {scanFor && (
        <form onSubmit={addScan} className="card row" data-testid="scan-form">
          <select
            aria-label="event type"
            value={scan.event_type}
            onChange={(e) =>
              setScan({ ...scan, event_type: e.target.value as typeof scan.event_type })
            }
          >
            {EVENT_TYPES.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
          <input
            aria-label="scan location"
            placeholder="Location"
            value={scan.location}
            onChange={(e) => setScan({ ...scan, location: e.target.value })}
            required
          />
          <button type="submit">Record</button>
          <button type="button" onClick={() => setScanFor('')}>
            Cancel
          </button>
        </form>
      )}
    </section>
  );
}
