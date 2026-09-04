const COLORS: Record<string, string> = {
  CREATED: '#6b7280',
  IN_TRANSIT: '#2563eb',
  OUT_FOR_DELIVERY: '#d97706',
  DELIVERED: '#16a34a',
  EXCEPTION: '#dc2626',
};

export function StatusBadge({ status }: { status: string }) {
  return (
    <span
      className="badge"
      style={{ background: COLORS[status] ?? '#6b7280' }}
      data-testid="status-badge"
    >
      {status.replace(/_/g, ' ')}
    </span>
  );
}
