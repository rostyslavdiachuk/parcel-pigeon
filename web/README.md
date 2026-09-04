# web

React + TypeScript + Vite SPA. Built to static files and served by Nginx, which
also proxies `/api` to the gateway.

- `src/pages/TrackPage.tsx` — public parcel tracking
- `src/pages/OpsPage.tsx` — operations dashboard (create shipment, add scans)
- `src/api.ts` — thin fetch client (`/api/*`, optional `x-api-key` from `VITE_API_KEY`)

## Local

```bash
npm install
npm test           # vitest + React Testing Library (fetch stubbed)
npm run lint
npm run dev        # http://localhost:5173, proxies /api to http://localhost:8000
```

Set `GATEWAY_URL` to point the dev proxy elsewhere.
