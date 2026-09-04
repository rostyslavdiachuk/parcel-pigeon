import { z } from 'zod';

const schema = z.object({
  PORT: z.coerce.number().default(8000),
  HOST: z.string().default('0.0.0.0'),
  SHIPMENTS_URL: z.string().url().default('http://shipments-service:8000'),
  TRACKING_URL: z.string().url().default('http://tracking-service:8002'),
  // When set, non-GET /api requests must carry this value in x-api-key.
  GATEWAY_API_KEY: z.string().optional(),
  LOG_LEVEL: z.string().default('info'),
});

export type Config = z.infer<typeof schema>;

export function loadConfig(env: NodeJS.ProcessEnv = process.env): Config {
  return schema.parse(env);
}
