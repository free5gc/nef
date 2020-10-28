package sbi

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *SBIServer) getPFDManagementEndpoints() []Endpoint {
	return []Endpoint{
		{
			Method:  strings.ToUpper("Get"),
			Pattern: "/:scsAsID/transactions",
			APIFunc: s.apiGetPFDManagementTransactions,
		},
		{
			Method:  strings.ToUpper("Post"),
			Pattern: "/:scsAsID/transactions",
			APIFunc: s.apiPostPFDManagementTransactions,
		},
		{
			Method:  strings.ToUpper("Get"),
			Pattern: "/:scsAsID/transactions/:transID",
			APIFunc: s.apiGetIndividualPFDManagementTransaction,
		},
		{
			Method:  strings.ToUpper("Put"),
			Pattern: "/:scsAsID/transactions/:transID",
			APIFunc: s.apiPutIndividualPFDManagementTransaction,
		},
		{
			Method:  strings.ToUpper("Delete"),
			Pattern: "/:scsAsID/transactions/:transID",
			APIFunc: s.apiDeleteIndividualPFDManagementTransaction,
		},
		{
			Method:  strings.ToUpper("Get"),
			Pattern: "/:scsAsID/transactions/:transID/applications/:appID",
			APIFunc: s.apiGetIndividualApplicationPFDManagement,
		},
		{
			Method:  strings.ToUpper("Delete"),
			Pattern: "/:scsAsID/transactions/:transID/applications/:appID",
			APIFunc: s.apiDeleteIndividualApplicationPFDManagement,
		},
		{
			Method:  strings.ToUpper("Put"),
			Pattern: "/:scsAsID/transactions/:transID/applications/:appID",
			APIFunc: s.apiPutIndividualApplicationPFDManagement,
		},
		{
			Method:  strings.ToUpper("Patch"),
			Pattern: "/:scsAsID/transactions/:transID/applications/:appID",
			APIFunc: s.apiPatchIndividualApplicationPFDManagement,
		},
	}
}

func (s *SBIServer) apiGetPFDManagementTransactions(ginCtx *gin.Context) {
}

func (s *SBIServer) apiPostPFDManagementTransactions(ginCtx *gin.Context) {
}

func (s *SBIServer) apiGetIndividualPFDManagementTransaction(ginCtx *gin.Context) {
}

func (s *SBIServer) apiPutIndividualPFDManagementTransaction(ginCtx *gin.Context) {
}

func (s *SBIServer) apiDeleteIndividualPFDManagementTransaction(ginCtx *gin.Context) {
}

func (s *SBIServer) apiGetIndividualApplicationPFDManagement(ginCtx *gin.Context) {
}

func (s *SBIServer) apiDeleteIndividualApplicationPFDManagement(ginCtx *gin.Context) {
}

func (s *SBIServer) apiPutIndividualApplicationPFDManagement(ginCtx *gin.Context) {
}

func (s *SBIServer) apiPatchIndividualApplicationPFDManagement(ginCtx *gin.Context) {
}
