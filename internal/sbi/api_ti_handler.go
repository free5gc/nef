package sbi

import (
	"strings"

	"github.com/gin-gonic/gin"
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
}

func (s *SBIServer) apiPostTrafficInfluenceSubscription(ginCtx *gin.Context) {
}

func (s *SBIServer) apiGetIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
}

func (s *SBIServer) apiPutIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
}

func (s *SBIServer) apiPatchIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
}

func (s *SBIServer) apiDeleteIndividualTrafficInfluenceSubscription(ginCtx *gin.Context) {
}
