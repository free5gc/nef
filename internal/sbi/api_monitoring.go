package sbi

import (
	"net/http"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/metrics/sbi"
	"github.com/gin-gonic/gin"
)

func (s *Server) getMonitoringEventRoutes() []Route {
	return []Route{
		{
			Method:  http.MethodGet,
			Pattern: "/:afID/subscriptions",
			APIFunc: s.apiGetMonitoringEventSubscriptions,
		},
		{
			Method:  http.MethodPost,
			Pattern: "/:afID/subscriptions",
			APIFunc: s.apiPostMonitoringEventSubscription,
		},
		{
			Method:  http.MethodGet,
			Pattern: "/:afID/subscriptions/:subID",
			APIFunc: s.apiGetIndividualMonitoringEventSubscription,
		},
		{
			Method:  http.MethodDelete,
			Pattern: "/:afID/subscriptions/:subID",
			APIFunc: s.apiDeleteIndividualMonitoringEventSubscription,
		},
	}
}

func (s *Server) apiGetMonitoringEventSubscriptions(gc *gin.Context) {
	s.Processor().GetMonitoringEventSubscriptions(gc, gc.Param("afID"))
}

func (s *Server) apiPostMonitoringEventSubscription(gc *gin.Context) {
	var monSub models.NefMonitoringEventSubscription
	reqBody, err := gc.GetRawData()
	if err != nil {
		logger.SBILog.Errorf("Get Request Body error: %+v", err)
		pd := openapi.ProblemDetailsSystemFailure(err.Error())
		gc.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		gc.JSON(http.StatusInternalServerError, pd)
		return
	}

	err = openapi.Deserialize(&monSub, reqBody, "application/json")
	if err != nil {
		logger.SBILog.Errorf("Deserialize Request Body error: %+v", err)
		pd := openapi.ProblemDetailsMalformedReqSyntax(err.Error())
		gc.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		gc.JSON(http.StatusBadRequest, pd)
		return
	}

	s.Processor().PostMonitoringEventSubscription(gc, gc.Param("afID"), &monSub)
}

func (s *Server) apiGetIndividualMonitoringEventSubscription(gc *gin.Context) {
	s.Processor().GetIndividualMonitoringEventSubscription(
		gc, gc.Param("afID"), gc.Param("subID"))
}

func (s *Server) apiDeleteIndividualMonitoringEventSubscription(gc *gin.Context) {
	s.Processor().DeleteIndividualMonitoringEventSubscription(
		gc, gc.Param("afID"), gc.Param("subID"))
}
