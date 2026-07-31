package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"gowoobro/global/config"
	"gowoobro/global/log"
	"gowoobro/router"

	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func Http() {
	log.Info().Str("service", "HTTP").Msg("Start Service")

	app := fiber.New(fiber.Config{
		BodyLimit:             500 * 1024 * 1024,
		Prefork:               false,
		CaseSensitive:         true,
		StrictRouting:         true,
		DisableStartupMessage: true,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
	})

	sites := strings.Join(config.Cors, ", ")
	if sites != "" {
		app.Use(cors.New(cors.Config{
			AllowOrigins:     sites,
			AllowCredentials: true,
			AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
			// Authorization 이 빠지면 관리자 화면이 통째로 막힌다. 이 헤더를 붙이는
			// 순간 요청이 프리플라이트 대상이 되는데, 허용 목록에 없으면 브라우저가
			// 본 요청을 아예 보내지 않는다. curl 은 CORS 를 지키지 않아 드러나지 않는다.
			AllowHeaders: "Origin,Content-Type,Accept,Content-Length,Accept-Language,Accept-Encoding,Connection,Access-Control-Allow-Origin,Authorization",
		}))
	}

	if config.Log.Web {
		app.Use(fiberzerolog.New(fiberzerolog.Config{
			Logger: log.Get(),
		}))
	}

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestCompression,
	}))

	// app.Static("/webdata", config.DocumentRoot)
	app.Static("/webdata", config.UploadPath)

	router.SetRouter(app)

	app.Get("/*", func(ctx *fiber.Ctx) error {
		return ctx.SendFile(fmt.Sprintf("./%v/index.html", config.DocumentRoot), true)
	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-c
		log.Info().Msg("Gracefully shutting down...")
		_ = app.Shutdown()
	}()

	if config.Mode == "develop" || !config.Tls.Use {
		if err := app.Listen(":" + config.Port); err != nil {
			log.Error().Msg(err.Error())
		}
	} else {
		cer, err := tls.LoadX509KeyPair(config.Tls.Cert, config.Tls.Key)
		if err != nil {
			log.Error().Msg("TLS error")
			log.Error().Msg(err.Error())
			return
		}

		cert := &tls.Config{Certificates: []tls.Certificate{cer}}

		ln, err := tls.Listen("tcp", ":"+config.Port, cert)
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		if err := app.Listener(ln); err != nil {
			log.Error().Msg(err.Error())
		}
	}
}
