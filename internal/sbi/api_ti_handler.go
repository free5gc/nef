package sbi

import (
	"strings"

	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/openapi/models"
)

func (s *SBIServer) getTrafficInfluenceEndpoints() []Endpoint {
	return []Endpoint{
		{
			Method:  strings.ToUpper("Get"),
			Pattern: "/:afID/subscriptions",
			APIFunc: s.apiGetTrafficInfluenceSubscription,
		},
		{
			Method:  strings.ToUpper("Post"),
			Pattern: "/:afID/subscriptions",
			APIFunc: s.apiPostTrafficInfluenceSubscription,
		},
		{
			Method:  strings.ToUpper("Get"),
			Pattern: "/:afID/subscriptions/:subscID",
			APIFunc: s.apiGetIndividualTrafficInfluenceSubscription,
		},
		{
			Method:  strings.ToUpper("Put"),
			Pattern: "/:afID/subscriptions/:subscID",
			APIFunc: s.apiPutIndividualTrafficInfluenceSubscription,
		},
		{
			Method:  strings.ToUpper("Patch"),
			Pattern: "/:afID/subscriptions/:subscID",
			APIFunc: s.apiPatchIndividualTrafficInfluenceSubscription,
		},
		{
			Method:  strings.ToUpper("Delete"),
			Pattern: "/:afID/subscriptions/:subscID",
			APIFunc: s.apiDeleteIndividualTrafficInfluenceSubscription,
		},
	}
}

func (s *SBIServer) apiGetTrafficInfluenceSubscription(ginCtx *gin.Context) {
	hdlRsp := s.processor.GetTrafficInfluenceSubscription(
		ginCtx.Param("afID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiPostTrafficInfluenceSubscription(ginCtx *gin.Context) {
	var tiSub models.TrafficInfluSub //TODO: change to AF-NEF TrafficInfluSub
	if err := s.getDataFromHttpRequestBody(ginCtx, &tiSub); err != nil {
		return
	}
	hdlRsp := s.processor.PostTrafficInfluenceSubscription(
		ginCtx.Param("afID"), &tiSub)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiGetIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
	hdlRsp := s.processor.GetIndividualTrafficInfluenceSubscription(
		ginCtx.Param("afID"), ginCtx.Param("subscID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiPutIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
	var tiSub models.TrafficInfluSub //TODO: change to AF-NEF TrafficInfluSub
	if err := s.getDataFromHttpRequestBody(ginCtx, &tiSub); err != nil {
		return
	}
	hdlRsp := s.processor.PutIndividualTrafficInfluenceSubscription(
		ginCtx.Param("afID"), ginCtx.Param("subscID"), &tiSub)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiPatchIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
	var tiSubPatch models.TrafficInfluDataPatch //TODO: change to AF-NEF TrafficInfluSubPatch
	if err := s.getDataFromHttpRequestBody(ginCtx, &tiSubPatch); err != nil {
		return
	}
	hdlRsp := s.processor.PatchIndividualTrafficInfluenceSubscription(
		ginCtx.Param("afID"), ginCtx.Param("subscID"), &tiSubPatch)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiDeleteIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
	hdlRsp := s.processor.DeleteIndividualTrafficInfluenceSubscription(
		ginCtx.Param("afID"), ginCtx.Param("subscID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}
