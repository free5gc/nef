package sbi

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/http2_util"
	"bitbucket.org/free5gc-team/logger_util"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/util"
)

type SBIServer struct {
	server    *http.Server
	router    *gin.Engine
}

func NewSBIServer() *SBIServer {
	s := &SBIServer{}
	s.init()

	addr := "0.0.0.0:12345" //TODO
	var err error
	if s.server, err = http2_util.NewServer(addr, util.NEF_LOG_PATH, s.router); err != nil {
		logger.InitLog.Errorf("Initialize HTTP server failed: %+v", err)
		return nil
	}

	return s
}

type Endpoint struct {
	Method  string
	Pattern string
	APIFunc gin.HandlerFunc
}

func (s *SBIServer) init() {
	s.router = logger_util.NewGinWithLogrus(logger.GinLog)

	endpoints := s.getTrafficInfluenceEndpoints()
	group := s.router.Group("/3gpp-traffic-influence/v1")
	applyEndpoints(group, endpoints)

	endpoints = s.getPFDManagementEndpoints()
	group = s.router.Group("/3gpp-pfd-management/v1")
	applyEndpoints(group, endpoints)

	endpoints = s.getOamEndpoints()
	group = s.router.Group("/nnef-oam/v1")
	applyEndpoints(group, endpoints)

	s.router.Use(cors.New(cors.Config{
		AllowMethods: []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "User-Agent",
			"Referrer", "Host", "Token", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowAllOrigins:  true,
		MaxAge:           86400,
	}))
}

func applyEndpoints(group *gin.RouterGroup, endpoints []Endpoint) {
	for _, endpoint := range endpoints {
		switch endpoint.Method {
		case "GET":
			group.GET(endpoint.Pattern, endpoint.APIFunc)
		case "POST":
			group.POST(endpoint.Pattern, endpoint.APIFunc)
		case "PUT":
			group.PUT(endpoint.Pattern, endpoint.APIFunc)
		case "PATCH":
			group.PATCH(endpoint.Pattern, endpoint.APIFunc)
		case "DELETE":
			group.DELETE(endpoint.Pattern, endpoint.APIFunc)
		}
	}
}

func (s *SBIServer) ListenAndServe(scheme string) error {
	if scheme == "http" {
		return s.server.ListenAndServe()
	} else if scheme == "https" {
		return s.server.ListenAndServeTLS(util.NEF_PEM_PATH, util.NEF_KEY_PATH) //use config
	}
	return fmt.Errorf("ListenAndServe Error: no support this scheme[%s]", scheme)
}

