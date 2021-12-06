package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/openapi/models"
)

func (s *Server) getTrafficInfluenceEndpoints() []Endpoint {
	return []Endpoint{
		{
			Method:  http.MethodGet,
			Pattern: "/:afID/subscriptions",
			APIFunc: s.apiGetTrafficInfluenceSubscription,
		},
		{
			Method:  http.MethodPost,
			Pattern: "/:afID/subscriptions",
			APIFunc: s.apiPostTrafficInfluenceSubscription,
		},
		{
			Method:  http.MethodGet,
			Pattern: "/:afID/subscriptions/:subscID",
			APIFunc: s.apiGetIndividualTrafficInfluenceSubscription,
		},
		{
			Method:  http.MethodPut,
			Pattern: "/:afID/subscriptions/:subscID",
			APIFunc: s.apiPutIndividualTrafficInfluenceSubscription,
		},
		{
			Method:  http.MethodPatch,
			Pattern: "/:afID/subscriptions/:subscID",
			APIFunc: s.apiPatchIndividualTrafficInfluenceSubscription,
		},
		{
			Method:  http.MethodDelete,
			Pattern: "/:afID/subscriptions/:subscID",
			APIFunc: s.apiDeleteIndividualTrafficInfluenceSubscription,
		},
	}
}

func (s *Server) apiGetTrafficInfluenceSubscription(ginCtx *gin.Context) {
	hdlRsp := s.Processor().GetTrafficInfluenceSubscription(
		ginCtx.Param("afID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp, false)
}

func (s *Server) apiPostTrafficInfluenceSubscription(ginCtx *gin.Context) {
	contentType, err := checkContentTypeIsJSON(ginCtx)
	if err != nil {
		return
	}

	var tiSub models.TrafficInfluSub
	if err := s.deserializeData(ginCtx, &tiSub, contentType); err != nil {
		return
	}

	hdlRsp := s.Processor().PostTrafficInfluenceSubscription(
		ginCtx.Param("afID"), &tiSub)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp, false)
}

func (s *Server) apiGetIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
	hdlRsp := s.Processor().GetIndividualTrafficInfluenceSubscription(
		ginCtx.Param("afID"), ginCtx.Param("subscID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp, false)
}

func (s *Server) apiPutIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
	contentType, err := checkContentTypeIsJSON(ginCtx)
	if err != nil {
		return
	}

	var tiSub models.TrafficInfluSub
	if err := s.deserializeData(ginCtx, &tiSub, contentType); err != nil {
		return
	}

	hdlRsp := s.Processor().PutIndividualTrafficInfluenceSubscription(
		ginCtx.Param("afID"), ginCtx.Param("subscID"), &tiSub)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp, false)
}

func (s *Server) apiPatchIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
	contentType, err := checkContentTypeIsJSON(ginCtx)
	if err != nil {
		return
	}

	var tiSubPatch models.TrafficInfluSubPatch
	if err := s.deserializeData(ginCtx, &tiSubPatch, contentType); err != nil {
		return
	}

	hdlRsp := s.Processor().PatchIndividualTrafficInfluenceSubscription(
		ginCtx.Param("afID"), ginCtx.Param("subscID"), &tiSubPatch)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp, false)
}

func (s *Server) apiDeleteIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
	hdlRsp := s.Processor().DeleteIndividualTrafficInfluenceSubscription(
		ginCtx.Param("afID"), ginCtx.Param("subscID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp, false)
}
