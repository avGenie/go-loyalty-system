package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	NetAddr     string `env:"RUN_ADDRESS"`
	DBConnect   string `env:"DATABASE_URI"`
	AccrualAddr string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	LogLevel    string `env:"LOG_LEVEL"`
}

func InitConfig() (config Config) {
	flag.StringVar(&config.NetAddr, "a", "localhost:8080", "net address host:port")
	flag.StringVar(&config.DBConnect, "d", "", "database credentials in format: host=host port=port user=myuser password=xxxx dbname=mydb sslmode=disable")
	flag.StringVar(&config.AccrualAddr, "r", "", "charge calculation system address")
	flag.StringVar(&config.LogLevel, "l", "info", "log level")
	flag.Parse()

	if err := env.Parse(&config); err != nil {
		panic(fmt.Errorf("error while parsing config: %w", err))
	}

	return
}
