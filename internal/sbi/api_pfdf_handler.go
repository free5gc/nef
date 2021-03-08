package sbi

import (
	"strings"

	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/openapi/models"
)

func (s *SBIServer) getPFDFEndpoints() []Endpoint {
	return []Endpoint{
		{
			Method:  strings.ToUpper("Get"),
			Pattern: "/applications",
			APIFunc: s.apiGetApplicationsPFD,
		},
		{
			Method:  strings.ToUpper("Get"),
			Pattern: "/applications/:appID",
			APIFunc: s.apiGetIndividualApplicationPFD,
		},
		{
			Method:  strings.ToUpper("Post"),
			Pattern: "/subscriptions",
			APIFunc: s.apiPostPFDSubscriptions,
		},
		{
			Method:  strings.ToUpper("Delete"),
			Pattern: "/subscriptions/:subscID",
			APIFunc: s.apiDeleteIndividualPFDSubscription,
		},
	}
}

func (s *SBIServer) apiGetApplicationsPFD(ginCtx *gin.Context) {
	//TODO: support URI query parameters: supported-features
	hdlRsp := s.processor.GetApplicationsPFD(ginCtx.QueryArray("application-ids"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiGetIndividualApplicationPFD(ginCtx *gin.Context) {
	//TODO: support URI query parameters: supported-features
	hdlRsp := s.processor.GetIndividualApplicationPFD(ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiPostPFDSubscriptions(ginCtx *gin.Context) {
	var pfdSubsc models.PfdSubscription
	if err := s.getDataFromHttpRequestBody(ginCtx, &pfdSubsc); err != nil {
		return
	}

	hdlRsp := s.processor.PostPFDSubscriptions(&pfdSubsc)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiDeleteIndividualPFDSubscription(ginCtx *gin.Context) {
	hdlRsp := s.processor.DeleteIndividualPFDSubscription(ginCtx.Param("subscID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}
