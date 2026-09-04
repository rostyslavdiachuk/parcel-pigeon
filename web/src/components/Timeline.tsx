import type { TrackEvent } from '../types';

export function Timeline({ events }: { events: TrackEvent[] }) {
  if (events.length === 0) {
    return <p className="muted">No scan events recorded yet.</p>;
  }
  return (
    <ol className="timeline" data-testid="timeline">
      {events.map((e, i) => (
        <li key={`${e.occurredAt}-${i}`}>
          <div className="timeline-dot" />
          <div>
            <strong>{e.eventType.replace(/_/g, ' ')}</strong>
            <div className="muted">
              {e.location} &middot; {new Date(e.occurredAt).toLocaleString()}
            </div>
          </div>
        </li>
      ))}
    </ol>
  );
}
