package sbi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/nef/internal/sbi/processor"
	nef_util "github.com/free5gc/nef/internal/util"
	"github.com/free5gc/nef/pkg/app"
	"github.com/free5gc/nef/pkg/factory"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/httpwrapper"
	logger_util "github.com/free5gc/util/logger"
	"github.com/free5gc/util/metrics"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	CorsConfigMaxAge = 86400
)

type nef interface {
	app.App
	Processor() *processor.Processor
}

type Server struct {
	nef

	httpServer *http.Server
	router     *gin.Engine
}

func NewServer(nef nef, tlsKeyLogPath string) (*Server, error) {
	s := &Server{
		nef: nef,
	}

	s.router = logger_util.NewGinWithLogrus(logger.GinLog)

	s.router.Use(metrics.InboundMetrics())

	// Callback endpoint: protected by OAuth2 middleware (callers must present
	// a valid nnef-callback Bearer token issued by NRF).
	callbackAuthCheck := nef_util.NewRouterAuthorizationCheck(models.Nrf_NFMgmt_ServiceName("nnef-callback"))
	callbackGroup := s.router.Group(factory.NefCallbackResUriPrefix)
	callbackGroup.Use(func(c *gin.Context) {
		callbackAuthCheck.Check(c, s.Context())
	})
	applyRoutes(callbackGroup, s.getCallbackRoutes())

	// All other route groups are mounted only when their service is declared in
	// ServiceList, and each group is protected by OAuth2 middleware.
	for _, service := range s.Config().ServiceList() {
		switch service.ServiceName {
		case factory.ServiceNefPfd:
			// nnef-pfdmanagement covers both the external AF-facing API
			// (/3gpp-pfd-management) and the SBI PFDF API (/nnef-pfdmanagement).
			authCheck := nef_util.NewRouterAuthorizationCheck(models.Nrf_NFMgmt_ServiceName_NNEF_PFDMANAGEMENT)

			pfdMngGroup := s.router.Group(factory.PfdMngResUriPrefix)
			pfdMngGroup.Use(func(c *gin.Context) {
				authCheck.Check(c, s.Context())
			})
			applyRoutes(pfdMngGroup, s.getPFDManagementRoutes())

			pfdFGroup := s.router.Group(factory.NefPfdMngResUriPrefix)
			pfdFGroup.Use(func(c *gin.Context) {
				authCheck.Check(c, s.Context())
			})
			applyRoutes(pfdFGroup, s.getPFDFRoutes())

		case factory.ServiceNefOam:
			authCheck := nef_util.NewRouterAuthorizationCheck(models.Nrf_NFMgmt_ServiceName(factory.ServiceNefOam))

			oamGroup := s.router.Group(factory.NefOamResUriPrefix)
			oamGroup.Use(func(c *gin.Context) {
				authCheck.Check(c, s.Context())
			})
			applyRoutes(oamGroup, s.getOamRoutes())

		case factory.ServiceTraffInflu:
			// 3gpp-traffic-influence is an AF-facing API (3GPP TS 29.522);
			authCheck := nef_util.NewRouterAuthorizationCheck(models.Nrf_NFMgmt_ServiceName_3GPP_TRAFFIC_INFLUENCE)
			tiGroup := s.router.Group(factory.TraffInfluResUriPrefix)
			tiGroup.Use(func(c *gin.Context) {
				authCheck.Check(c, s.Context())
			})
			applyRoutes(tiGroup, s.getTrafficInfluenceRoutes())

		case factory.ServiceNefMonitoringEvent:
			// 3gpp-monitoring-event is an AF-facing API (3GPP TS 29.122, reused by TS 29.522
			// for 5GS);
			authCheck := nef_util.NewRouterAuthorizationCheck(models.Nrf_NFMgmt_ServiceName_3GPP_MONITORING_EVENT)
			monGroup := s.router.Group(factory.MonitoringEventResUriPrefix)
			monGroup.Use(func(c *gin.Context) {
				authCheck.Check(c, s.Context())
			})
			applyRoutes(monGroup, s.getMonitoringEventRoutes())
		}
	}

	s.router.Use(cors.New(cors.Config{
		AllowMethods: []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{
			"Origin", "Content-Length", "Content-Type", "User-Agent",
			"Referrer", "Host", "Token", "X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowAllOrigins:  true,
		MaxAge:           CorsConfigMaxAge,
	}))

	bindAddr := s.Config().SbiBindingAddr()
	logger.SBILog.Infof("Binding addr: [%s]", bindAddr)
	var err error
	if s.httpServer, err = httpwrapper.NewHttp2Server(bindAddr, tlsKeyLogPath, s.router); err != nil {
		logger.InitLog.Errorf("Initialize HTTP server failed: %+v", err)
		return nil, err
	}

	return s, nil
}

func (s *Server) Run(wg *sync.WaitGroup) error {
	wg.Add(1)
	go s.startServer(wg)
	return nil
}

func (s *Server) Terminate() {
	const defaultShutdownTimeout time.Duration = 2 * time.Second

	if s.httpServer != nil {
		logger.SBILog.Infof("Stop SBI server (listen on %s)", s.httpServer.Addr)
		toCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		if err := s.httpServer.Shutdown(toCtx); err != nil {
			logger.SBILog.Errorf("Could not close SBI server: %#v", err)
		}
	}
}

func (s *Server) startServer(wg *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.SBILog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
			s.Terminate()
		}

		wg.Done()
	}()

	logger.SBILog.Infof("Start SBI server (listen on %s)", s.httpServer.Addr)

	var err error

	scheme := s.Config().SbiScheme()
	switch scheme {
	case "http":
		err = s.httpServer.ListenAndServe()
	case "https":
		// TODO: use config file to config path
		err = s.httpServer.ListenAndServeTLS(s.Config().GetCertPemPath(), s.Config().GetCertKeyPath())
	default:
		err = fmt.Errorf("scheme [%s] is not supported", scheme)
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.SBILog.Errorf("SBI server error: %+v", err)
	}
	logger.SBILog.Warnf("SBI server (listen on %s) stopped", s.httpServer.Addr)
}
