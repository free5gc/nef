package processor

import (
	"net/http"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi/models"
)

func (p *Processor) GetPFDManagementTransactions(scsAsID string) *HandlerResponse {
	logger.PFDManageLog.Infof("GetPFDManagementTransactions - scsAsID[%s]", scsAsID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PostPFDManagementTransactions(scsAsID string, pfdMng *models.PfdManagement) *HandlerResponse {
	logger.PFDManageLog.Infof("PostPFDManagementTransactions - scsAsID[%s]", scsAsID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) GetIndividualPFDManagementTransaction(scsAsID, transID string) *HandlerResponse {
	logger.PFDManageLog.Infof("GetIndividualPFDManagementTransaction - scsAsID[%s], transID[%s]", scsAsID, transID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PutIndividualPFDManagementTransaction(scsAsID, transID string,
	pfdMng *models.PfdManagement) *HandlerResponse {
	logger.PFDManageLog.Infof("PutIndividualPFDManagementTransaction - scsAsID[%s], transID[%s]", scsAsID, transID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) DeleteIndividualPFDManagementTransaction(scsAsID, transID string) *HandlerResponse {
	logger.PFDManageLog.Infof("DeleteIndividualPFDManagementTransaction - scsAsID[%s], transID[%s]", scsAsID, transID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) GetIndividualApplicationPFDManagement(scsAsID, transID, appID string) *HandlerResponse {
	logger.PFDManageLog.Infof("GetIndividualApplicationPFDManagement - scsAsID[%s], transID[%s], appID[%s]",
		scsAsID, transID, appID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) DeleteIndividualApplicationPFDManagement(scsAsID, transID, appID string) *HandlerResponse {
	logger.PFDManageLog.Infof("DeleteIndividualApplicationPFDManagement - scsAsID[%s], transID[%s], appID[%s]",
		scsAsID, transID, appID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PutIndividualApplicationPFDManagement(scsAsID, transID, appID string,
	pfdData *models.PfdData) *HandlerResponse {
	logger.PFDManageLog.Infof("PutIndividualApplicationPFDManagement - scsAsID[%s], transID[%s], appID[%s]",
		scsAsID, transID, appID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PatchIndividualApplicationPFDManagement(scsAsID, transID, appID string,
	pfdData *models.PfdData) *HandlerResponse {
	logger.PFDManageLog.Infof("PatchIndividualApplicationPFDManagement - scsAsID[%s], transID[%s], appID[%s]",
		scsAsID, transID, appID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}
