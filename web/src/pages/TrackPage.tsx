import { useState } from 'react';
import { api, ApiError } from '../api';
import type { TrackState } from '../types';
import { StatusBadge } from '../components/StatusBadge';
import { Timeline } from '../components/Timeline';

export function TrackPage() {
  const [value, setValue] = useState('');
  const [state, setState] = useState<TrackState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trackingNumber = value.trim();
    if (!trackingNumber) return;
    setLoading(true);
    setError(null);
    setState(null);
    try {
      setState(await api.track(trackingNumber));
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 404
          ? `No parcel found for "${trackingNumber}"`
          : 'Something went wrong looking that up.',
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <section>
      <h2>Track a parcel</h2>
      <form onSubmit={onSubmit} className="row">
        <input
          aria-label="tracking number"
          placeholder="PP-DEMO0001"
          value={value}
          onChange={(e) => setValue(e.target.value)}
        />
        <button type="submit" disabled={loading}>
          {loading ? 'Looking…' : 'Track'}
        </button>
      </form>

      {error && (
        <p role="alert" className="error">
          {error}
        </p>
      )}

      {state && (
        <article className="card">
          <header className="row space">
            <div>
              <h3>{state.trackingNumber}</h3>
              <p className="muted">
                {state.origin} → {state.destination}
              </p>
            </div>
            <StatusBadge status={state.status} />
          </header>
          {state.eta && <p className="muted">ETA: {new Date(state.eta).toLocaleString()}</p>}
          <Timeline events={state.events} />
        </article>
      )}
    </section>
  );
}
