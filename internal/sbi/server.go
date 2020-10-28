package sbi

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/http2_util"
	"bitbucket.org/free5gc-team/http_wrapper"
	"bitbucket.org/free5gc-team/logger_util"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/processor"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/models"
)

type SBIServer struct {
	cfg       *factory.Config
	server    *http.Server
	router    *gin.Engine
	processor *processor.Processor
}

func NewSBIServer(nefCfg *factory.Config, proc *processor.Processor) *SBIServer {
	s := &SBIServer{cfg: nefCfg, processor: proc}
	s.init()

	bindAddr := s.cfg.GetBindingAddr()
	logger.SBILog.Infof("Binding addr: [%s]", bindAddr)
	var err error
	if s.server, err = http2_util.NewServer(bindAddr, factory.NEF_LOG_PATH, s.router); err != nil {
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
		//TODO: use config file to config path
		return s.server.ListenAndServeTLS(factory.NEF_PEM_PATH, factory.NEF_KEY_PATH)
	}
	return fmt.Errorf("ListenAndServe Error: no support this scheme[%s]", scheme)
}

func (s *SBIServer) getDataFromHttpRequestBody(ginCtx *gin.Context, data interface{}) error {
	reqBody, err := ginCtx.GetRawData()
	if err != nil {
		logger.SBILog.Errorf("Get Request Body error: %+v", err)
		problemDetail := models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
			Cause:  "SYSTEM_FAILURE",
		}
		ginCtx.JSON(http.StatusInternalServerError, problemDetail)
		return err
	}

	err = openapi.Deserialize(data, reqBody, "application/json")
	if err != nil {
		logger.SBILog.Errorf("Deserialize Request Body error: %+v", err)
		detail := "[Request Body] " + err.Error()
		problemDetail := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: detail,
		}
		logger.SBILog.Errorln(problemDetail)
		ginCtx.JSON(http.StatusBadRequest, problemDetail)
		return err
	}

	return nil
}

func (s *SBIServer) buildAndSendHttpResponse(ginCtx *gin.Context, hdlRsp *processor.HandlerResponse) {
	rsp := http_wrapper.NewResponse(hdlRsp.Status, hdlRsp.Headers, hdlRsp.Body)
	rspBody, err := openapi.Serialize(rsp.Body, "application/json")
	if err != nil {
		logger.SBILog.Errorln(err)
		problemDetails := models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Cause:  "SYSTEM_FAILURE",
			Detail: err.Error(),
		}
		ginCtx.JSON(http.StatusInternalServerError, problemDetails)
	} else {
		ginCtx.Data(rsp.Status, "application/json", rspBody)
	}
}
