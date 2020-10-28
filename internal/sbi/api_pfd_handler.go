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
	hdlRsp := s.processor.GetPFDManagementTransactions(
		ginCtx.Param("scsAsID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiPostPFDManagementTransactions(ginCtx *gin.Context) {
	//var pfdManag models.PfdManagement
	//if err := s.getDataFromHttpRequestBody(ginCtx, &pfdManag); err != nil {
	//	return
	//}
	hdlRsp := s.processor.PostPFDManagementTransactions(
		ginCtx.Param("scsAsID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiGetIndividualPFDManagementTransaction(ginCtx *gin.Context) {
	hdlRsp := s.processor.GetIndividualPFDManagementTransaction(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiPutIndividualPFDManagementTransaction(ginCtx *gin.Context) {
	//var pfdManag models.PfdManagement
	//if err := s.getDataFromHttpRequestBody(ginCtx, &pfdManag); err != nil {
	//	return
	//}
	hdlRsp := s.processor.PutIndividualPFDManagementTransaction(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiDeleteIndividualPFDManagementTransaction(ginCtx *gin.Context) {
	hdlRsp := s.processor.DeleteIndividualPFDManagementTransaction(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiGetIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	hdlRsp := s.processor.GetIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiDeleteIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	hdlRsp := s.processor.DeleteIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiPutIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	//var pfdManag models.PfdManagement
	//if err := s.getDataFromHttpRequestBody(ginCtx, &pfdManag); err != nil {
	//	return
	//}
	hdlRsp := s.processor.PutIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *SBIServer) apiPatchIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	//var pfdManag models.PfdManagement
	//if err := s.getDataFromHttpRequestBody(ginCtx, &pfdManag); err != nil {
	//	return
	//}
	hdlRsp := s.processor.PatchIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}
