# gateway

Edge service. Fastify reverse proxy in front of `shipments-service` and
`tracking-service`.

- `src/app.ts` — `buildApp(config)` factory (routing, CORS, stub auth, health)
- `src/plugins/metrics.ts` — RED metrics + `/metrics`
- `src/config.ts` — env validation with zod

Routing:

| Incoming | Upstream |
|----------|----------|
| `/api/shipments/*` | `SHIPMENTS_URL` + `/shipments/*` |
| `/api/track/*` | `TRACKING_URL` + `/track/*` |
| `/api/notifications` | `TRACKING_URL` + `/notifications` |

Stub auth: if `GATEWAY_API_KEY` is set, non-GET `/api/*` needs `x-api-key`.

## Local

```bash
npm install
npm test          # vitest, uses an in-process stub upstream
npm run lint
npm run dev       # tsx watch
```

Config: `PORT`, `HOST`, `SHIPMENTS_URL`, `TRACKING_URL`, `GATEWAY_API_KEY`, `LOG_LEVEL`.
