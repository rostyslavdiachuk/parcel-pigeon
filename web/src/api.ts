import type { CreateShipmentInput, ScanEvent, Shipment, TrackState } from './types';

const API_KEY = import.meta.env.VITE_API_KEY as string | undefined;

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body) headers.set('Content-Type', 'application/json');
  if (API_KEY && init.method && init.method !== 'GET') headers.set('x-api-key', API_KEY);

  const res = await fetch(`/api${path}`, { ...init, headers });
  if (!res.ok) {
    let detail = res.statusText;
    try {
      const body = await res.json();
      detail = body.detail ?? body.error ?? detail;
    } catch {
      /* keep statusText */
    }
    throw new ApiError(res.status, detail);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export const api = {
  listShipments: () => request<Shipment[]>('/shipments'),
  getShipment: (id: string) => request<Shipment & { scans: ScanEvent[] }>(`/shipments/${id}`),
  createShipment: (input: CreateShipmentInput) =>
    request<Shipment>('/shipments', { method: 'POST', body: JSON.stringify(input) }),
  addScan: (id: string, input: { event_type: string; location: string; note?: string }) =>
    request<ScanEvent>(`/shipments/${id}/scans`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  track: (trackingNumber: string) => request<TrackState>(`/track/${trackingNumber}`),
};
