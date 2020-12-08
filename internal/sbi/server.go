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
	"bitbucket.org/free5gc-team/nef/internal/util"
	"bitbucket.org/free5gc-team/openapi"
)

const (
	CORS_CONFIG_MAXAGE = 86400
)

type SBIServer struct {
	cfg       *factory.Config
	server    *http.Server
	router    *gin.Engine
	processor *processor.Processor
}

func NewSBIServer(nefCfg *factory.Config, proc *processor.Processor) *SBIServer {
	s := &SBIServer{cfg: nefCfg, processor: proc}

	s.router = logger_util.NewGinWithLogrus(logger.GinLog)

	endpoints := s.getTrafficInfluenceEndpoints()
	group := s.router.Group(factory.TRAFF_INFLU_RES_URI_PREFIX)
	applyEndpoints(group, endpoints)

	endpoints = s.getPFDManagementEndpoints()
	group = s.router.Group(factory.PFD_MNG_RES_URI_PREFIX)
	applyEndpoints(group, endpoints)

	endpoints = s.getPFDFEndpoints()
	group = s.router.Group(factory.NEF_PFD_MNG_RES_URI_PREFIX)
	applyEndpoints(group, endpoints)

	endpoints = s.getOamEndpoints()
	group = s.router.Group(factory.NEF_OAM_RES_URI_PREFIX)
	applyEndpoints(group, endpoints)

	s.router.Use(cors.New(cors.Config{
		AllowMethods: []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "User-Agent",
			"Referrer", "Host", "Token", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowAllOrigins:  true,
		MaxAge:           CORS_CONFIG_MAXAGE,
	}))

	bindAddr := s.cfg.GetSbiBindingAddr()
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

func (s *SBIServer) buildAndSendHttpResponse(ginCtx *gin.Context, hdlRsp *processor.HandlerResponse) {
	if hdlRsp.Status == 0 {
		// No Response to send
		return
	}

	rsp := http_wrapper.NewResponse(hdlRsp.Status, hdlRsp.Headers, hdlRsp.Body)

	buildHttpResponseHeader(ginCtx, rsp)

	if rspBody, err := openapi.Serialize(rsp.Body, "application/json"); err != nil {
		logger.SBILog.Errorln(err)
		ginCtx.JSON(http.StatusInternalServerError, util.ProblemDetailsSystemFailure(err.Error()))
	} else {
		ginCtx.Data(rsp.Status, "application/json", rspBody)
	}
}

func buildHttpResponseHeader(ginCtx *gin.Context, rsp *http_wrapper.Response) {
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
