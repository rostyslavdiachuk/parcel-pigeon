import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OpsPage } from '../pages/OpsPage';
import type { Shipment } from '../types';

const shipment: Shipment = {
  id: 's1',
  tracking_number: 'PP-DEMO0002',
  recipient_name: 'Alan Turing',
  recipient_email: 'alan@example.com',
  origin: 'Manchester',
  destination: 'Berlin',
  status: 'IN_TRANSIT',
  eta: null,
  created_at: '2026-09-01T00:00:00Z',
  updated_at: '2026-09-01T00:00:00Z',
};

afterEach(() => vi.unstubAllGlobals());

describe('OpsPage', () => {
  it('lists shipments from the API', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({ ok: true, status: 200, json: async () => [shipment] }) as Response),
    );
    render(<OpsPage />);

    expect(await screen.findByText('PP-DEMO0002')).toBeInTheDocument();
    expect(screen.getByTestId('status-badge')).toHaveTextContent('IN TRANSIT');
  });

  it('POSTs a new shipment with the form values', async () => {
    const calls: Array<{ url: string; init?: RequestInit }> = [];
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string, init?: RequestInit) => {
        calls.push({ url, init });
        return {
          ok: true,
          status: init?.method === 'POST' ? 201 : 200,
          json: async () => (init?.method === 'POST' ? shipment : []),
        } as Response;
      }),
    );
    render(<OpsPage />);

    await userEvent.type(screen.getByLabelText('recipient name'), 'Grace Hopper');
    await userEvent.type(screen.getByLabelText('recipient email'), 'grace@example.com');
    await userEvent.type(screen.getByLabelText('origin'), 'New York');
    await userEvent.type(screen.getByLabelText('destination'), 'Boston');
    await userEvent.click(screen.getByRole('button', { name: 'Create shipment' }));

    const post = calls.find((c) => c.init?.method === 'POST');
    expect(post?.url).toBe('/api/shipments');
    expect(JSON.parse(String(post?.init?.body))).toEqual({
      recipient_name: 'Grace Hopper',
      recipient_email: 'grace@example.com',
      origin: 'New York',
      destination: 'Boston',
    });
  });

  it('records a scan for the selected shipment', async () => {
    const calls: Array<{ url: string; init?: RequestInit }> = [];
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string, init?: RequestInit) => {
        calls.push({ url, init });
        return { ok: true, status: 200, json: async () => [shipment] } as Response;
      }),
    );
    render(<OpsPage />);
    await screen.findByText('PP-DEMO0002');

    await userEvent.click(screen.getByRole('button', { name: 'Add scan' }));
    const form = screen.getByTestId('scan-form');
    await userEvent.selectOptions(within(form).getByLabelText('event type'), 'OUT_FOR_DELIVERY');
    await userEvent.type(within(form).getByLabelText('scan location'), 'Berlin Depot');
    await userEvent.click(within(form).getByRole('button', { name: 'Record' }));

    const post = calls.find((c) => c.init?.method === 'POST');
    expect(post?.url).toBe('/api/shipments/s1/scans');
    expect(JSON.parse(String(post?.init?.body))).toEqual({
      event_type: 'OUT_FOR_DELIVERY',
      location: 'Berlin Depot',
    });
  });
});
