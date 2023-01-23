package main

import (
	"context"
	"github.com/coocood/freecache"
	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigdotenv"
	cache "github.com/gitsight/go-echo-cache"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	Port string `env:"ECHO_HTTP_PORT"`
	Root string `env:"ECHO_HTTP_ROOT"`
}

var Version string = "1.1.0"

var err error

var Cfg Config

func main() {

	loader := aconfig.LoaderFor(&Cfg, aconfig.Config{
		SkipFlags:          false,
		AllowUnknownEnvs:   true,
		AllowUnknownFields: true,
		SkipEnv:            false,
		//	EnvPrefix:          "ECHO_HTTP",
		FileDecoders: map[string]aconfig.FileDecoder{
			".env": aconfigdotenv.New(),
		},
		Files: []string{".env"},
	})

	if err = loader.Load(); err != nil {
		return
	}
	if Cfg.Port == "" {
		Cfg.Port = "8080"
	}
	if Cfg.Root == "" {
		Cfg.Root = "./www"
	}

	logger := zerolog.New(os.Stdout).With().
		Str("app", "echo-http-server"). //.Timestamp().
		Logger()
	logger.Info().Str("root", Cfg.Root).Str("port", Cfg.Port).Str("version", "v"+Version).Msg("Starting Echo HTTP Server")
	freeCache := freecache.NewCache(1024 * 2) // Pre-allocated cache of 2KB)

	app := echo.New()
	app.HideBanner = true
	app.HidePort = true
	app.Use(cache.New(&cache.Config{}, freeCache))
	app.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogMethod:   true,

		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {

			if len(v.Headers["X-Real-Ip"]) > 0 {
				v.RemoteIP = v.Headers["X-Real-Ip"][0]
			}

			logger.Info().
				Str("path", v.URI).
				Int("status", v.Status).
				Dur("latency", v.Latency).
				Str("ip", v.RemoteIP).
				Str("method", v.Method).
				//				Int64("cache-LookupCount", freeCache.LookupCount()).
				Msg("request")

			return nil
		},
	}))
	app.Use(middleware.Static(Cfg.Root))
	// Start server
	go func() {
		if err := app.Start(":" + Cfg.Port); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Msg("shutting down the server")
		}
	}()
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Fatal().Msg("Graceful shutdown ...")

	if err := app.Shutdown(ctx); err != nil {
		logger.Fatal().Msg(err.Error())
	}

}
