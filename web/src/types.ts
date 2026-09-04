export type ShipmentStatus =
  'CREATED' | 'IN_TRANSIT' | 'OUT_FOR_DELIVERY' | 'DELIVERED' | 'EXCEPTION';

export interface Shipment {
  id: string;
  tracking_number: string;
  recipient_name: string;
  recipient_email: string;
  origin: string;
  destination: string;
  status: ShipmentStatus;
  eta: string | null;
  created_at: string;
  updated_at: string;
}

export interface ScanEvent {
  id: string;
  event_type: string;
  location: string;
  note: string | null;
  occurred_at: string;
}

export interface TrackEvent {
  status: string;
  eventType: string;
  location: string;
  occurredAt: string;
}

export interface TrackState {
  trackingNumber: string;
  status: ShipmentStatus;
  previousStatus: string;
  origin: string;
  destination: string;
  recipientName: string;
  eta: string;
  events: TrackEvent[];
  updatedAt: string;
}

export interface CreateShipmentInput {
  recipient_name: string;
  recipient_email: string;
  origin: string;
  destination: string;
}

export const EVENT_TYPES = [
  'PICKED_UP',
  'IN_TRANSIT',
  'ARRIVED_AT_FACILITY',
  'OUT_FOR_DELIVERY',
  'DELIVERED',
  'DELIVERY_FAILED',
  'RETURNED',
] as const;
