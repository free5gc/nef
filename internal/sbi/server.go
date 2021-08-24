package sbi

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/processor"
	"bitbucket.org/free5gc-team/nef/internal/util"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/util/httpwrapper"
	logger_util "bitbucket.org/free5gc-team/util/logger"
)

const (
	CorsConfigMaxAge = 86400
)

type Server struct {
	cfg        *factory.Config
	httpServer *http.Server
	router     *gin.Engine
	processor  *processor.Processor
}

func NewServer(nefCfg *factory.Config, proc *processor.Processor) *Server {
	s := &Server{cfg: nefCfg, processor: proc}

	s.router = logger_util.NewGinWithLogrus(logger.GinLog)

	endpoints := s.getTrafficInfluenceEndpoints()
	group := s.router.Group(factory.TraffInfluResUriPrefix)
	applyEndpoints(group, endpoints)

	endpoints = s.getPFDManagementEndpoints()
	group = s.router.Group(factory.PfdMngResUriPrefix)
	applyEndpoints(group, endpoints)

	endpoints = s.getPFDFEndpoints()
	group = s.router.Group(factory.NefPfdMngResUriPrefix)
	applyEndpoints(group, endpoints)

	endpoints = s.getOamEndpoints()
	group = s.router.Group(factory.NefOamResUriPrefix)
	applyEndpoints(group, endpoints)

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

	bindAddr := s.cfg.GetSbiBindingAddr()
	logger.SBILog.Infof("Binding addr: [%s]", bindAddr)
	var err error
	if s.httpServer, err = httpwrapper.NewHttp2Server(bindAddr, factory.NefDefaultKeyLogPath, s.router); err != nil {
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

func (s *Server) getDataFromHttpRequestBody(ginCtx *gin.Context, data interface{}) error {
	reqBody, err := ginCtx.GetRawData()
	if err != nil {
		logger.SBILog.Errorf("Get Request Body error: %+v", err)
		ginCtx.JSON(http.StatusInternalServerError,
			util.ProblemDetailsSystemFailure(err.Error()))
		return err
	}

	err = openapi.Deserialize(data, reqBody, "application/json")
	if err != nil {
		logger.SBILog.Errorf("Deserialize Request Body error: %+v", err)
		ginCtx.JSON(http.StatusBadRequest,
			util.ProblemDetailsMalformedReqSyntax(err.Error()))
		return err
	}

	return nil
}

func (s *Server) Run(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go s.startServer(wg)
	return nil
}

func (s *Server) Stop(ctx context.Context, wg *sync.WaitGroup) {
	if s.httpServer != nil {
		logger.SBILog.Infof("Stop SBI server (listen on %s)", s.httpServer.Addr)
		if err := s.httpServer.Close(); err != nil {
			logger.SBILog.Errorf("Could not close SBI server: %#v", err)
		}
	}
}

func (s *Server) startServer(wg *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.SBILog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}

		wg.Done()
	}()

	logger.SBILog.Infof("Start SBI server (listen on %s)", s.httpServer.Addr)

	var err error
	scheme := s.cfg.GetSbiScheme()
	if scheme == "http" {
		err = s.httpServer.ListenAndServe()
	} else if scheme == "https" {
		// TODO: use config file to config path
		err = s.httpServer.ListenAndServeTLS(factory.NefDefaultPemPath, factory.NefDefaultKeyPath)
	} else {
		err = fmt.Errorf("No support this scheme[%s]", scheme)
	}

	if err != nil {
		logger.SBILog.Errorf("SBI server error: %+v", err)
	}
	logger.SBILog.Infof("SBI server (listen on %s) stopped", s.httpServer.Addr)
}

func (s *Server) buildAndSendHttpResponse(ginCtx *gin.Context, hdlRsp *processor.HandlerResponse) {
	if hdlRsp.Status == 0 {
		// No Response to send
		return
	}

	rsp := httpwrapper.NewResponse(hdlRsp.Status, hdlRsp.Headers, hdlRsp.Body)

	buildHttpResponseHeader(ginCtx, rsp)

	if rspBody, err := openapi.Serialize(rsp.Body, "application/json"); err != nil {
		logger.SBILog.Errorln(err)
		ginCtx.JSON(http.StatusInternalServerError, util.ProblemDetailsSystemFailure(err.Error()))
	} else {
		ginCtx.Data(rsp.Status, "application/json", rspBody)
	}
}

func buildHttpResponseHeader(ginCtx *gin.Context, rsp *httpwrapper.Response) {
	for k, v := range rsp.Header {
		// Concatenate all values of the Header with ','
		allValues := ""
		for i, vv := range v {
			if i == 0 {
				allValues += vv
			} else {
				allValues += "," + vv
			}
		}
		ginCtx.Header(k, allValues)
	}
}
