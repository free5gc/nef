package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/openapi/models"
)

func (s *Server) getPFDManagementEndpoints() []Endpoint {
	return []Endpoint{
		{
			Method:  http.MethodGet,
			Pattern: "/:scsAsID/transactions",
			APIFunc: s.apiGetPFDManagementTransactions,
		},
		{
			Method:  http.MethodPost,
			Pattern: "/:scsAsID/transactions",
			APIFunc: s.apiPostPFDManagementTransactions,
		},
		{
			Method:  http.MethodDelete,
			Pattern: "/:scsAsID/transactions",
			APIFunc: s.apiDeletePFDManagementTransactions,
		},
		{
			Method:  http.MethodGet,
			Pattern: "/:scsAsID/transactions/:transID",
			APIFunc: s.apiGetIndividualPFDManagementTransaction,
		},
		{
			Method:  http.MethodPut,
			Pattern: "/:scsAsID/transactions/:transID",
			APIFunc: s.apiPutIndividualPFDManagementTransaction,
		},
		{
			Method:  http.MethodDelete,
			Pattern: "/:scsAsID/transactions/:transID",
			APIFunc: s.apiDeleteIndividualPFDManagementTransaction,
		},
		{
			Method:  http.MethodGet,
			Pattern: "/:scsAsID/transactions/:transID/applications/:appID",
			APIFunc: s.apiGetIndividualApplicationPFDManagement,
		},
		{
			Method:  http.MethodDelete,
			Pattern: "/:scsAsID/transactions/:transID/applications/:appID",
			APIFunc: s.apiDeleteIndividualApplicationPFDManagement,
		},
		{
			Method:  http.MethodPut,
			Pattern: "/:scsAsID/transactions/:transID/applications/:appID",
			APIFunc: s.apiPutIndividualApplicationPFDManagement,
		},
		{
			Method:  http.MethodPatch,
			Pattern: "/:scsAsID/transactions/:transID/applications/:appID",
			APIFunc: s.apiPatchIndividualApplicationPFDManagement,
		},
	}
}

func (s *Server) apiGetPFDManagementTransactions(ginCtx *gin.Context) {
	hdlRsp := s.Processor().GetPFDManagementTransactions(
		ginCtx.Param("scsAsID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiPostPFDManagementTransactions(ginCtx *gin.Context) {
	var pfdMng models.PfdManagement
	if err := s.deserializeData(ginCtx, &pfdMng); err != nil {
		return
	}

	hdlRsp := s.Processor().PostPFDManagementTransactions(
		ginCtx.Param("scsAsID"), &pfdMng)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiDeletePFDManagementTransactions(ginCtx *gin.Context) {
	hdlRsp := s.Processor().DeletePFDManagementTransactions(
		ginCtx.Param("scsAsID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiGetIndividualPFDManagementTransaction(ginCtx *gin.Context) {
	hdlRsp := s.Processor().GetIndividualPFDManagementTransaction(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiPutIndividualPFDManagementTransaction(ginCtx *gin.Context) {
	var pfdMng models.PfdManagement
	if err := s.deserializeData(ginCtx, &pfdMng); err != nil {
		return
	}

	hdlRsp := s.Processor().PutIndividualPFDManagementTransaction(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), &pfdMng)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiDeleteIndividualPFDManagementTransaction(ginCtx *gin.Context) {
	hdlRsp := s.Processor().DeleteIndividualPFDManagementTransaction(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiGetIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	hdlRsp := s.Processor().GetIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiDeleteIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	hdlRsp := s.Processor().DeleteIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiPutIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	var pfdData models.PfdData
	if err := s.deserializeData(ginCtx, &pfdData); err != nil {
		return
	}

	hdlRsp := s.Processor().PutIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"), &pfdData)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiPatchIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	var pfdData models.PfdData
	if err := s.deserializeData(ginCtx, &pfdData); err != nil {
		return
	}

	hdlRsp := s.Processor().PatchIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"), &pfdData)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}
