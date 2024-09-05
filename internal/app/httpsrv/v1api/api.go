package v1api

import (
	"fmt"
	"net/http"

	"github.com/SergeyParamoshkin/alerts/internal/app/httpsrv"
	"github.com/SergeyParamoshkin/alerts/internal/app/httpsrv/v1api/docs"
	_ "github.com/SergeyParamoshkin/alerts/internal/app/httpsrv/v1api/docs"
	"github.com/SergeyParamoshkin/alerts/internal/tel"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
)

var _ httpsrv.API = &API{}

type API struct {
	router    *chi.Mux
	logger    *zap.Logger
	config    *Config
	telemetry *tel.Telemetry

	ticketService TicketService

	promHandler http.Handler
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func (a *API) Version() string {
	return "v1"
}

func (a *API) GenSwaggerJSON(host, basePath string, schemes []string) string {
	swaggerInfo := *docs.SwaggerInfo
	swaggerInfo.Host = host
	swaggerInfo.BasePath = basePath
	swaggerInfo.Schemes = schemes

	return swaggerInfo.ReadDoc()
}

//	@title			Watwell alerts API
//	@version		v1.0
//	@description	This is a server

//	@contact.name	API Support

//	@host		localhost:8080
//	@BasePath	/v1

// @externalDocs.description	wiki.yandex.ru
// @externalDocs.url			https://wiki.yandex.ru/
func New(params Params) (Result, error) {
	logger := params.Logger.Named("api")
	config := params.Config

	telemetry := params.Telemetry
	telemetryRegistry := telemetry.Registry()

	api := &API{
		router:        chi.NewRouter(),
		logger:        logger,
		config:        config,
		telemetry:     telemetry,
		ticketService: params.TicketService,
		promHandler: promhttp.InstrumentMetricHandler(
			telemetryRegistry,
			promhttp.HandlerFor(telemetryRegistry, promhttp.HandlerOpts{}),
		),
	}

	swaggerJSON := api.GenSwaggerJSON(
		config.SwaggerUI.Host,
		config.SwaggerUI.BasePath,
		config.SwaggerUI.Schemes,
	)

	api.router.Route(fmt.Sprintf("/%s", api.Version()), func(router chi.Router) {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST"},
		}))

		router.Route("/ticket", func(r chi.Router) {
			r.Post("/list", api.ticketList)
		})

		router.Get("/info", api.info)

		router.Route("/swagger", func(router chi.Router) {
			router.Get("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte(swaggerJSON)); err != nil {
					logger.Error("failed to write swagger yaml", zap.Error(err))

					return
				}
			})

			router.Get("/*", httpSwagger.Handler(
				httpSwagger.URL(fmt.Sprintf("%s/swagger/swagger.json", config.SwaggerUI.BasePath)),
			))
		})
	})

	api.router.Route("/metrics", func(r chi.Router) {
		r.Get("/", api.metricsHandler)
	})

	if config.Debug {
		api.router.Route("/debug", func(r chi.Router) {
			r.Mount("/", chiMiddleware.Profiler())
		})
	}

	return Result{
		API: api,
	}, nil
}
