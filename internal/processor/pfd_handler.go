package processor

import (
	"fmt"
	"net/http"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/util"
	"bitbucket.org/free5gc-team/openapi/models"
)

const (
	PFD_ERR_NO_PFD_DATA        = "Absent of PfdManagement.PfdDatas"
	PFD_ERR_NO_PFD             = "Absent of PfdData.Pfds"
	PFD_ERR_NO_EXTERNAL_APP_ID = "Absent of PfdData.ExternalAppID"
	PFD_ERR_NO_PFD_ID          = "Absent of Pfd.PfdID"
	PFD_ERR_NO_FLOW_IDENT      = "One of FlowDescriptions, Urls or DomainNames should be provided"
)

func (p *Processor) GetPFDManagementTransactions(scsAsID string) *HandlerResponse {
	logger.PFDManageLog.Infof("GetPFDManagementTransactions - scsAsID[%s]", scsAsID)

	afCtx := p.nefCtx.GetAfCtx(scsAsID)
	if afCtx == nil {
		problemDetails := util.ProblemDetailsDataNotFound("Given AF is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	var pfdMngs []models.PfdManagement
	for _, afPfdTrans := range afCtx.GetAllPfdTrans() {
		pfdMng := models.PfdManagement{
			Self:     genPfdManagementURI(p.cfg.GetSbiUri(), scsAsID, afPfdTrans.GetTransID()),
			PfdDatas: make(map[string]models.PfdData),
		}

		rspCode, rspBody := p.consumer.UdrSrv.AppDataPfdsGet(afPfdTrans.GetExtAppIDs())
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		for _, pfdDataForApp := range *(rspBody.(*[]models.PfdDataForApp)) {
			pfdData := convertPfdDataForAppToPfdData(&pfdDataForApp)
			pfdData.Self = genPfdDataURI(p.cfg.GetSbiUri(), scsAsID, afPfdTrans.GetTransID(), pfdData.ExternalAppId)
			pfdMng.PfdDatas[pfdData.ExternalAppId] = *pfdData
		}
		pfdMngs = append(pfdMngs, pfdMng)
	}

	return &HandlerResponse{http.StatusOK, nil, &pfdMngs}
}

func (p *Processor) PostPFDManagementTransactions(scsAsID string, pfdMng *models.PfdManagement) *HandlerResponse {
	logger.PFDManageLog.Infof("PostPFDManagementTransactions - scsAsID[%s]", scsAsID)

	// TODO: Authorize the AF

	problemDetails := validatePfdManagement(pfdMng, p.nefCtx)
	if problemDetails != nil {
		if problemDetails.Status == http.StatusInternalServerError {
			return &HandlerResponse{http.StatusInternalServerError, nil, &pfdMng.PfdReports}
		} else {
			return &HandlerResponse{int(problemDetails.Status), nil, problemDetails}
		}
	}

	afCtx := p.nefCtx.GetAfCtx(scsAsID)
	if afCtx == nil {
		afCtx = p.nefCtx.NewAfCtx(scsAsID)
	}
	afTrans := p.nefCtx.NewAfPfdTrans(afCtx)

	for appID, pfdData := range pfdMng.PfdDatas {
		afTrans.AddExtAppID(appID)
		pfdDataForApp := convertPfdDataToPfdDataForApp(&pfdData)
		rspCode, _ := p.consumer.UdrSrv.AppDataPfdsAppIdPut(appID, pfdDataForApp)
		if rspCode != http.StatusCreated {
			delete(pfdMng.PfdDatas, appID)
			addPfdReport(pfdMng, appID, models.FailureCode_MALFUNCTION)
		} else {
			pfdData.Self = genPfdDataURI(p.cfg.GetSbiUri(), scsAsID, afTrans.GetTransID(), appID)
			pfdMng.PfdDatas[appID] = pfdData
		}
	}
	if len(pfdMng.PfdDatas) == 0 {
		// The PFDs for all applications were not created successfully.
		// PfdReport is included with detailed information.
		return &HandlerResponse{http.StatusInternalServerError, nil, &pfdMng.PfdReports}
	}

	afCtx.AddPfdTrans(afTrans)
	p.nefCtx.AddAfCtx(afCtx)

	pfdMng.Self = genPfdManagementURI(p.cfg.GetSbiUri(), scsAsID, afTrans.GetTransID())
	// Not mandated by TS 29.122 v1.15.3
	hdrs := make(map[string][]string)
	util.AddLocationheader(hdrs, pfdMng.Self)

	return &HandlerResponse{http.StatusCreated, hdrs, pfdMng}
}

func (p *Processor) DeletePFDManagementTransactions(scsAsID string) *HandlerResponse {
	logger.PFDManageLog.Infof("DeletePFDManagementTransactions - scsAsID[%s]", scsAsID)

	afCtx := p.nefCtx.GetAfCtx(scsAsID)
	if afCtx == nil {
		problemDetails := util.ProblemDetailsDataNotFound("Given AF is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	for _, afPfdTrans := range afCtx.GetAllPfdTrans() {
		for _, extAppID := range afPfdTrans.GetExtAppIDs() {
			rspCode, rspBody := p.consumer.UdrSrv.AppDataPfdsAppIdDelete(extAppID)
			if rspCode != http.StatusNoContent {
				return &HandlerResponse{rspCode, nil, rspBody}
			}
		}
		afCtx.DeletePfdTrans(afPfdTrans.GetTransID())
	}

	// TODO: Remove AfCtx if its subscriptions and transactions are both empty

	return &HandlerResponse{http.StatusNoContent, nil, nil}
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

	_, err := p.nefCtx.GetPfdTransWithAppID(scsAsID, transID, appID)
	if err != nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(err.Error())}
	}

	rspCode, rspBody := p.consumer.UdrSrv.AppDataPfdsAppIdGet(appID)
	if rspCode != http.StatusOK {
		return &HandlerResponse{rspCode, nil, rspBody}
	}
	pfdData := convertPfdDataForAppToPfdData(rspBody.(*models.PfdDataForApp))
	pfdData.Self = genPfdDataURI(p.cfg.GetSbiUri(), scsAsID, transID, appID)

	return &HandlerResponse{http.StatusOK, nil, pfdData}
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

func convertPfdDataForAppToPfdData(pfdDataForApp *models.PfdDataForApp) *models.PfdData {
	pfdData := &models.PfdData{
		ExternalAppId: pfdDataForApp.ApplicationId,
		Pfds:          make(map[string]models.Pfd),
	}
	for _, pfdContent := range pfdDataForApp.Pfds {
		var pfd models.Pfd
		pfd.PfdId = pfdContent.PfdId
		pfd.DomainNames = pfdContent.DomainNames
		pfd.FlowDescriptions = pfdContent.FlowDescriptions
		pfd.Urls = pfdContent.Urls
		pfdData.Pfds[pfd.PfdId] = pfd
	}
	return pfdData
}

func convertPfdDataToPfdDataForApp(pfdData *models.PfdData) *models.PfdDataForApp {
	pfdDataForApp := &models.PfdDataForApp{
		ApplicationId: pfdData.ExternalAppId,
	}
	for _, pfd := range pfdData.Pfds {
		var pfdContent models.PfdContent
		pfdContent.PfdId = pfd.PfdId
		pfdContent.FlowDescriptions = pfd.FlowDescriptions
		pfdContent.Urls = pfd.Urls
		pfdContent.DomainNames = pfd.DomainNames
		pfdDataForApp.Pfds = append(pfdDataForApp.Pfds, pfdContent)
	}
	return pfdDataForApp
}

func genPfdManagementURI(sbiURI, afID, transID string) string {
	// E.g. https://localhost:29505/3gpp-pfd-management/v1/{afID}/transactions/{transID}
	return fmt.Sprintf("%s%s/%s/transactions/%s",
		sbiURI, factory.PFD_MNG_RES_URI_PREFIX, afID, transID)
}

func genPfdDataURI(sbiURI, afID, transID, appID string) string {
	// E.g. https://localhost:29505/3gpp-pfd-management/v1/{afID}/transactions/{transID}/applications/{appID}
	return fmt.Sprintf("%s%s/%s/transactions/%s/applications/%s",
		sbiURI, factory.PFD_MNG_RES_URI_PREFIX, afID, transID, appID)
}

func validatePfdManagement(pfdMng *models.PfdManagement, nefCtx *context.NefContext) *models.ProblemDetails {
	pfdMng.PfdReports = make(map[string]models.PfdReport)

	if len(pfdMng.PfdDatas) == 0 {
		return util.ProblemDetailsDataNotFound(PFD_ERR_NO_PFD_DATA)
	}

	for appID, pfdData := range pfdMng.PfdDatas {
		if appID == "" {
			return util.ProblemDetailsDataNotFound(PFD_ERR_NO_EXTERNAL_APP_ID)
		}

		if len(pfdData.Pfds) == 0 {
			return util.ProblemDetailsDataNotFound(PFD_ERR_NO_PFD)
		}

		// Check whether the received external Application Identifier(s) are already provisioned
		if nefCtx.IsAppIDExisted(appID) {
			delete(pfdMng.PfdDatas, appID)
			addPfdReport(pfdMng, appID, models.FailureCode_APP_ID_DUPLICATED)
		}

		for pfdID, pfd := range pfdData.Pfds {
			if pfdID == "" {
				return util.ProblemDetailsDataNotFound(PFD_ERR_NO_PFD_ID)
			}
			if len(pfd.FlowDescriptions) == 0 && len(pfd.Urls) == 0 && len(pfd.DomainNames) == 0 {
				return util.ProblemDetailsDataNotFound(PFD_ERR_NO_FLOW_IDENT)
			}
		}
	}

	if len(pfdMng.PfdDatas) == 0 {
		// The PFDs for all applications were not created successfully.
		// PfdReport is included with detailed information.
		return util.ProblemDetailsSystemFailure("None of the PFDs were created")
	} else {
		return nil
	}
}

func addPfdReport(pfdMng *models.PfdManagement, appID string, failureCode models.FailureCode) {
	if pfdReport, ok := pfdMng.PfdReports[string(failureCode)]; ok {
		pfdReport.ExternalAppIds = append(pfdReport.ExternalAppIds, appID)
	} else {
		pfdMng.PfdReports[string(failureCode)] = models.PfdReport{
			ExternalAppIds: []string{appID},
			FailureCode:    failureCode,
		}
	}
}
