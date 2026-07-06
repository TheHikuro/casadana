package config

import (
	"fmt"
	"net/url"

	"github.com/caarlos0/env/v11"
)

type DBConfig struct {
	Host     string `env:"POSTGRES_HOST,required"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER,required"`
	Password string `env:"POSTGRES_PASSWORD,required"`
	Name     string `env:"POSTGRES_DB,required"`
}

func (d DBConfig) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:     d.Name,
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

type Config struct {
	Port             int    `env:"PORT" envDefault:"8080"`
	LogLevel         string `env:"LOG_LEVEL" envDefault:"info"`
	DB               DBConfig
	JWTSecret        string `env:"JWT_SECRET,required"`
	ResendKey        string `env:"RESEND_API_KEY,required"`
	MailFrom         string `env:"MAIL_FROM,required"`
	WebOrigin        string `env:"WEB_ORIGIN,required"`
	AdminNotifyEmail string `env:"ADMIN_NOTIFY_EMAIL,required"`
	MigrateOnBoot    bool   `env:"MIGRATE_ON_BOOT" envDefault:"true"`
	CookieSecure     bool   `env:"COOKIE_SECURE" envDefault:"true"`
}

func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return c, nil
}
