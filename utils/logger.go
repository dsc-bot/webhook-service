package utils

import (
	// "github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func Configure(sentryDsn *string, json bool, logLevel zapcore.Level) error {
	// if sentryDsn != nil {
	// 	if err := sentry.Init(sentry.ClientOptions{
	// 		Dsn: *sentryDsn,
	// 	}); err != nil {
	// 		return nil, err
	// 	}
	// }

	if json {
		loggerConfig := zap.NewProductionConfig()
		loggerConfig.Level.SetLevel(logLevel)

		logger, err := loggerConfig.Build(
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
			// zap.WrapCore(ZapSentryAdapter(EnvironmentProduction)),
		)
		Logger = logger
		return err
	} else {
		loggerConfig := zap.NewDevelopmentConfig()
		loggerConfig.Level.SetLevel(logLevel)
		loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

		logger, err := loggerConfig.Build(zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
		Logger = logger
		return err
	}
}
