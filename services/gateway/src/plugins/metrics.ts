import fp from 'fastify-plugin';
import type { FastifyInstance } from 'fastify';
import { collectDefaultMetrics, Counter, Histogram, Registry } from 'prom-client';

/**
 * RED metrics + a Prometheus scrape endpoint. Route label is the Fastify route
 * template (e.g. "/api/shipments/*") so cardinality stays bounded.
 */
export const metricsPlugin = fp(async (app: FastifyInstance) => {
  const registry = new Registry();
  collectDefaultMetrics({ register: registry, prefix: 'gateway_' });

  const requests = new Counter({
    name: 'http_requests_total',
    help: 'HTTP requests handled',
    labelNames: ['method', 'route', 'status'],
    registers: [registry],
  });
  const latency = new Histogram({
    name: 'http_request_duration_seconds',
    help: 'HTTP request latency',
    labelNames: ['method', 'route'],
    registers: [registry],
  });

  app.addHook('onResponse', async (req, reply) => {
    const route = req.routeOptions?.url ?? req.url;
    const seconds = reply.elapsedTime / 1000;
    latency.labels(req.method, route).observe(seconds);
    requests.labels(req.method, route, String(reply.statusCode)).inc();
  });

  app.get('/metrics', async (_req, reply) => {
    reply.header('Content-Type', registry.contentType);
    return registry.metrics();
  });
});
