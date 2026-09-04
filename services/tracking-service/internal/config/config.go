package config

import "os"

// Config is the full runtime configuration, read from environment variables.
type Config struct {
	HTTPAddr       string
	RedisAddr      string
	RabbitURL      string
	Exchange       string
	Queue          string
	ShipmentsURL   string
	SMTPAddr       string
	MailFrom       string
	EnableConsumer bool
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// Load builds a Config from TRACKING_* environment variables with sane
// docker-compose defaults.
func Load() Config {
	return Config{
		HTTPAddr:       getenv("TRACKING_HTTP_ADDR", ":8002"),
		RedisAddr:      getenv("TRACKING_REDIS_ADDR", "redis:6379"),
		RabbitURL:      getenv("TRACKING_RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		Exchange:       getenv("TRACKING_EXCHANGE", "parcelpigeon"),
		Queue:          getenv("TRACKING_QUEUE", "tracking.projector"),
		ShipmentsURL:   getenv("TRACKING_SHIPMENTS_URL", "http://shipments-service:8000"),
		SMTPAddr:       getenv("TRACKING_SMTP_ADDR", "mailhog:1025"),
		MailFrom:       getenv("TRACKING_MAIL_FROM", "no-reply@parcelpigeon.test"),
		EnableConsumer: getenv("TRACKING_ENABLE_CONSUMER", "true") == "true",
	}
}
