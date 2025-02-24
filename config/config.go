package config

import (
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	LogLevel zapcore.Level `env:"LOG_LEVEL" envDefault:"debug"`
	JsonLogs bool          `env:"JSON_LOGS" envDefault:"false"`

	Rabbit struct {
		Url   string `env:"URL,required"`
		Queue string `env:"QUEUE,required" envDefault:"webhook_queue"`
	} `envPrefix:"RABBIT_"`
}

var Conf Config

func Parse() {
	var err error
	if _, err = os.Stat(".env"); err == nil {
		err = godotenv.Load(".env")
		if err != nil {
			panic(err)
		}
	}

	if err := env.Parse(&Conf); err != nil {
		panic(err)
	}
}
