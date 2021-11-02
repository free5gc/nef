package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/openapi/models"
)

func (s *Server) getPFDFEndpoints() []Endpoint {
	return []Endpoint{
		{
			Method:  http.MethodGet,
			Pattern: "/applications",
			APIFunc: s.apiGetApplicationsPFD,
		},
		{
			Method:  http.MethodGet,
			Pattern: "/applications/:appID",
			APIFunc: s.apiGetIndividualApplicationPFD,
		},
		{
			Method:  http.MethodPost,
			Pattern: "/subscriptions",
			APIFunc: s.apiPostPFDSubscriptions,
		},
		{
			Method:  http.MethodDelete,
			Pattern: "/subscriptions/:subscID",
			APIFunc: s.apiDeleteIndividualPFDSubscription,
		},
	}
}

func (s *Server) apiGetApplicationsPFD(ginCtx *gin.Context) {
	//TODO: support URI query parameters: supported-features
	hdlRsp := s.Processor().GetApplicationsPFD(ginCtx.QueryArray("application-ids"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiGetIndividualApplicationPFD(ginCtx *gin.Context) {
	//TODO: support URI query parameters: supported-features
	hdlRsp := s.Processor().GetIndividualApplicationPFD(ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiPostPFDSubscriptions(ginCtx *gin.Context) {
	var pfdSubsc models.PfdSubscription
	if err := s.deserializeData(ginCtx, &pfdSubsc); err != nil {
		return
	}

	hdlRsp := s.Processor().PostPFDSubscriptions(&pfdSubsc)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiDeleteIndividualPFDSubscription(ginCtx *gin.Context) {
	hdlRsp := s.Processor().DeleteIndividualPFDSubscription(ginCtx.Param("subscID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}
