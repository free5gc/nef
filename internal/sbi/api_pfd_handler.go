package sbi

import (
	"strings"

	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/openapi/models"
)

func (s *Server) getPFDManagementEndpoints() []Endpoint {
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
			Method:  strings.ToUpper("Delete"),
			Pattern: "/:scsAsID/transactions",
			APIFunc: s.apiDeletePFDManagementTransactions,
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

func (s *Server) apiGetPFDManagementTransactions(ginCtx *gin.Context) {
	hdlRsp := s.processor.GetPFDManagementTransactions(
		ginCtx.Param("scsAsID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiPostPFDManagementTransactions(ginCtx *gin.Context) {
	var pfdMng models.PfdManagement
	if err := s.deserializeData(ginCtx, &pfdMng); err != nil {
		return
	}

	hdlRsp := s.processor.PostPFDManagementTransactions(
		ginCtx.Param("scsAsID"), &pfdMng)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiDeletePFDManagementTransactions(ginCtx *gin.Context) {
	hdlRsp := s.processor.DeletePFDManagementTransactions(
		ginCtx.Param("scsAsID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiGetIndividualPFDManagementTransaction(ginCtx *gin.Context) {
	hdlRsp := s.processor.GetIndividualPFDManagementTransaction(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiPutIndividualPFDManagementTransaction(ginCtx *gin.Context) {
	var pfdMng models.PfdManagement
	if err := s.deserializeData(ginCtx, &pfdMng); err != nil {
		return
	}

	hdlRsp := s.processor.PutIndividualPFDManagementTransaction(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), &pfdMng)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiDeleteIndividualPFDManagementTransaction(ginCtx *gin.Context) {
	hdlRsp := s.processor.DeleteIndividualPFDManagementTransaction(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiGetIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	hdlRsp := s.processor.GetIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiDeleteIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	hdlRsp := s.processor.DeleteIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"))

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiPutIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	var pfdData models.PfdData
	if err := s.deserializeData(ginCtx, &pfdData); err != nil {
		return
	}

	hdlRsp := s.processor.PutIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"), &pfdData)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}

func (s *Server) apiPatchIndividualApplicationPFDManagement(ginCtx *gin.Context) {
	var pfdData models.PfdData
	if err := s.deserializeData(ginCtx, &pfdData); err != nil {
		return
	}

	hdlRsp := s.processor.PatchIndividualApplicationPFDManagement(
		ginCtx.Param("scsAsID"), ginCtx.Param("transID"), ginCtx.Param("appID"), &pfdData)

	s.buildAndSendHttpResponse(ginCtx, hdlRsp)
}
