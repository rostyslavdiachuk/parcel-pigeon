import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { TrackPage } from '../pages/TrackPage';
import type { TrackState } from '../types';

const sample: TrackState = {
  trackingNumber: 'PP-DEMO0001',
  status: 'DELIVERED',
  previousStatus: 'OUT_FOR_DELIVERY',
  origin: 'London',
  destination: 'Paris',
  recipientName: 'Ada Lovelace',
  eta: '2026-09-01T10:00:00Z',
  updatedAt: '2026-09-01T10:00:00Z',
  events: [
    {
      status: 'IN_TRANSIT',
      eventType: 'PICKED_UP',
      location: 'London Hub',
      occurredAt: '2026-08-30T10:00:00Z',
    },
    {
      status: 'DELIVERED',
      eventType: 'DELIVERED',
      location: 'Paris',
      occurredAt: '2026-09-01T09:00:00Z',
    },
  ],
};

afterEach(() => vi.unstubAllGlobals());

function stubFetch(impl: (url: string) => { ok: boolean; status: number; body: unknown }) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string) => {
      const { ok, status, body } = impl(url);
      return { ok, status, json: async () => body, statusText: 'x' } as Response;
    }),
  );
}

describe('TrackPage', () => {
  it('renders the timeline for a found parcel', async () => {
    stubFetch(() => ({ ok: true, status: 200, body: sample }));
    render(<TrackPage />);

    await userEvent.type(screen.getByLabelText('tracking number'), 'PP-DEMO0001');
    await userEvent.click(screen.getByRole('button', { name: 'Track' }));

    expect(await screen.findByTestId('timeline')).toBeInTheDocument();
    expect(screen.getByTestId('status-badge')).toHaveTextContent('DELIVERED');
    expect(screen.getAllByRole('listitem')).toHaveLength(2);
  });

  it('shows a not-found message on a 404', async () => {
    stubFetch(() => ({ ok: false, status: 404, body: { detail: 'nope' } }));
    render(<TrackPage />);

    await userEvent.type(screen.getByLabelText('tracking number'), 'PP-NOPE');
    await userEvent.click(screen.getByRole('button', { name: 'Track' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('No parcel found for "PP-NOPE"');
  });

  it('calls the track endpoint with the entered number', async () => {
    stubFetch(() => ({ ok: true, status: 200, body: sample }));
    render(<TrackPage />);

    await userEvent.type(screen.getByLabelText('tracking number'), 'PP-DEMO0001');
    await userEvent.click(screen.getByRole('button', { name: 'Track' }));
    await screen.findByTestId('timeline');

    expect(fetch).toHaveBeenCalledWith('/api/track/PP-DEMO0001', expect.anything());
  });
});
