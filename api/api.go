package api

import (
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/web"
	static "github.com/TwiN/gatus/v5/web"
	"github.com/TwiN/health"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberfs "github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/redirect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	router *fiber.App
}

func New(cfg *config.Config) *API {
	api := &API{}
	if cfg.Web == nil {
		log.Println("[api.New] nil web config passed as parameter. This should only happen in tests. Using default web configuration")
		cfg.Web = web.GetDefaultConfig()
	}
	api.router = api.createRouter(cfg)
	return api
}

func (a *API) Router() *fiber.App {
	return a.router
}

func (a *API) createRouter(cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Printf("[api.ErrorHandler] %s", err.Error())
			return fiber.DefaultErrorHandler(c, err)
		},
		ReadBufferSize: cfg.Web.ReadBufferSize,
		Network:        fiber.NetworkTCP,
	})
	if os.Getenv("ENVIRONMENT") == "dev" {
		app.Use(cors.New(cors.Config{
			AllowOrigins:     "http://localhost:8081",
			AllowCredentials: true,
		}))
	}
	// Middlewares
	app.Use(recover.New())
	app.Use(compress.New())
	// Define metrics handler, if necessary
	if cfg.Metrics {
		metricsHandler := promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
			DisableCompression: true,
		}))
		app.Get("/metrics", adaptor.HTTPHandler(metricsHandler))
	}
	// Define main router
	apiRouter := app.Group("/api")
	////////////////////////
	// UNPROTECTED ROUTES //
	////////////////////////
	unprotectedAPIRouter := apiRouter.Group("/")
	unprotectedAPIRouter.Get("/v1/config", ConfigHandler{securityConfig: cfg.Security}.GetConfig)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/health/badge.svg", HealthBadge)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/health/badge.shields", HealthBadgeShields)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/uptimes/:duration/badge.svg", UptimeBadge)
	unprotectedAPIRouter.Get("/v1/endpoints/:key/response-times/:duration/badge.svg", ResponseTimeBadge(cfg))
	unprotectedAPIRouter.Get("/v1/endpoints/:key/response-times/:duration/chart.svg", ResponseTimeChart)
	// SPA
	app.Get("/", SinglePageApplication(cfg.UI))
	app.Get("/endpoints/:name", SinglePageApplication(cfg.UI))
	// Health endpoint
	healthHandler := health.Handler().WithJSON(true)
	app.Get("/health", func(c *fiber.Ctx) error {
		statusCode, body := healthHandler.GetResponseStatusCodeAndBody()
		return c.Status(statusCode).Send(body)
	})
	// Everything else falls back on static content
	app.Use(redirect.New(redirect.Config{
		Rules: map[string]string{
			"/index.html": "/",
		},
		StatusCode: 301,
	}))
	staticFileSystem, err := fs.Sub(static.FileSystem, static.RootPath)
	if err != nil {
		panic(err)
	}
	app.Use("/", fiberfs.New(fiberfs.Config{
		Root:   http.FS(staticFileSystem),
		Index:  "index.html",
		Browse: true,
	}))
	//////////////////////
	// PROTECTED ROUTES //
	//////////////////////
	// ORDER IS IMPORTANT: all routes applied AFTER the security middleware will require authn
	protectedAPIRouter := apiRouter.Group("/")
	if cfg.Security != nil {
		if err := cfg.Security.RegisterHandlers(app); err != nil {
			panic(err)
		}
		if err := cfg.Security.ApplySecurityMiddleware(protectedAPIRouter); err != nil {
			panic(err)
		}
	}
	protectedAPIRouter.Get("/v1/endpoints/statuses", EndpointStatuses(cfg))
	protectedAPIRouter.Get("/v1/endpoints/:key/statuses", EndpointStatus)
	return app
}
