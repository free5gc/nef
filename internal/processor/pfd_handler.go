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
	DetailNoAF       = "Given AF is not existed"
	DetailNoPfdData  = "Absent of PfdManagement.PfdDatas"
	DetailNoPfd      = "Absent of PfdData.Pfds"
	DetailNoExtAppID = "Absent of PfdData.ExternalAppID"
	DetailNoPfdID    = "Absent of Pfd.PfdID"
	DetailNoPfdInfo  = "One of FlowDescriptions, Urls or DomainNames should be provided"
)

func (p *Processor) GetPFDManagementTransactions(scsAsID string) *HandlerResponse {
	logger.PFDManageLog.Infof("GetPFDManagementTransactions - scsAsID[%s]", scsAsID)

	afCtx := p.nefCtx.GetAfCtx(scsAsID)
	if afCtx == nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(DetailNoAF)}
	}

	var pfdMngs []models.PfdManagement
	for _, afPfdTrans := range afCtx.GetAllPfdTrans() {
		pfdMng, rsp := p.buildPfdManagement(scsAsID, afPfdTrans)
		if pfdMng == nil {
			return rsp
		}
		pfdMngs = append(pfdMngs, *pfdMng)
	}

	return &HandlerResponse{http.StatusOK, nil, &pfdMngs}
}

func (p *Processor) PostPFDManagementTransactions(scsAsID string, pfdMng *models.PfdManagement) *HandlerResponse {
	logger.PFDManageLog.Infof("PostPFDManagementTransactions - scsAsID[%s]", scsAsID)

	// TODO: Authorize the AF

	if problemDetails := validatePfdManagement(scsAsID, "-1", pfdMng, p.nefCtx); problemDetails != nil {
		if problemDetails.Status == http.StatusInternalServerError {
			return &HandlerResponse{http.StatusInternalServerError, nil, &pfdMng.PfdReports}
		} else {
			return &HandlerResponse{int(problemDetails.Status), nil, problemDetails}
		}
	}

	afCtx := p.nefCtx.GetAfCtx(scsAsID)
	if afCtx == nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(DetailNoAF)}
	}
	afTrans := p.nefCtx.NewAfPfdTrans(afCtx)

	pfdNotifyContext := p.notifier.PfdChangeNotifier.NewPfdNotifyContext()
	defer pfdNotifyContext.FlushNotifications()

	for appID, pfdData := range pfdMng.PfdDatas {
		afTrans.AddExtAppID(appID)
		pfdDataForApp := convertPfdDataToPfdDataForApp(&pfdData)
		if pfdReport := p.storePfdDataToUDR(appID, pfdDataForApp); pfdReport != nil {
			delete(pfdMng.PfdDatas, appID)
			addPfdReport(pfdMng, pfdReport)
		} else {
			pfdData.Self = genPfdDataURI(p.cfg.GetSbiUri(), scsAsID, afTrans.GetTransID(), appID)
			pfdMng.PfdDatas[appID] = pfdData
			pfdNotifyContext.AddNotification(appID, &models.PfdChangeNotification{
				ApplicationId: appID,
				Pfds:          pfdDataForApp.Pfds,
			})
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

	return &HandlerResponse{http.StatusCreated, nil, pfdMng}
}

func (p *Processor) DeletePFDManagementTransactions(scsAsID string) *HandlerResponse {
	logger.PFDManageLog.Infof("DeletePFDManagementTransactions - scsAsID[%s]", scsAsID)

	afCtx := p.nefCtx.GetAfCtx(scsAsID)
	if afCtx == nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(DetailNoAF)}
	}

	pfdNotifyContext := p.notifier.PfdChangeNotifier.NewPfdNotifyContext()
	defer pfdNotifyContext.FlushNotifications()

	for _, afPfdTrans := range afCtx.GetAllPfdTrans() {
		for _, extAppID := range afPfdTrans.GetExtAppIDs() {
			if rsp := p.deletePfdDataFromUDR(extAppID); rsp != nil {
				return rsp
			}
			pfdNotifyContext.AddNotification(extAppID, &models.PfdChangeNotification{
				ApplicationId: extAppID,
				RemovalFlag:   true,
			})
		}
		afCtx.DeletePfdTrans(afPfdTrans.GetTransID())
	}

	// TODO: Remove AfCtx if its subscriptions and transactions are both empty

	return &HandlerResponse{http.StatusNoContent, nil, nil}
}

func (p *Processor) GetIndividualPFDManagementTransaction(scsAsID, transID string) *HandlerResponse {
	logger.PFDManageLog.Infof("GetIndividualPFDManagementTransaction - scsAsID[%s], transID[%s]", scsAsID, transID)

	_, afPfdTrans, err := p.nefCtx.GetAfCtxAndPfdTransWithTransID(scsAsID, transID)
	if err != nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(err.Error())}
	}

	pfdMng, rsp := p.buildPfdManagement(scsAsID, afPfdTrans)
	if pfdMng == nil {
		return rsp
	}

	return &HandlerResponse{http.StatusOK, nil, pfdMng}
}

func (p *Processor) PutIndividualPFDManagementTransaction(scsAsID, transID string,
	pfdMng *models.PfdManagement) *HandlerResponse {

	logger.PFDManageLog.Infof("PutIndividualPFDManagementTransaction - scsAsID[%s], transID[%s]", scsAsID, transID)

	// TODO: Authorize the AF

	if problemDetails := validatePfdManagement(scsAsID, transID, pfdMng, p.nefCtx); problemDetails != nil {
		if problemDetails.Status == http.StatusInternalServerError {
			return &HandlerResponse{http.StatusInternalServerError, nil, &pfdMng.PfdReports}
		} else {
			return &HandlerResponse{int(problemDetails.Status), nil, problemDetails}
		}
	}

	_, afPfdTrans, err := p.nefCtx.GetAfCtxAndPfdTransWithTransID(scsAsID, transID)
	if err != nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(err.Error())}
	}

	pfdNotifyContext := p.notifier.PfdChangeNotifier.NewPfdNotifyContext()
	defer pfdNotifyContext.FlushNotifications()

	// Delete PfdDataForApps in UDR with appID absent in new PfdManagement
	deprecatedAppIDs := []string{}
	for _, appID := range afPfdTrans.GetExtAppIDs() {
		if _, exist := pfdMng.PfdDatas[appID]; !exist {
			deprecatedAppIDs = append(deprecatedAppIDs, appID)
		}
	}
	for _, appID := range deprecatedAppIDs {
		if rsp := p.deletePfdDataFromUDR(appID); rsp != nil {
			return rsp
		}
		pfdNotifyContext.AddNotification(appID, &models.PfdChangeNotification{
			ApplicationId: appID,
			RemovalFlag:   true,
		})
	}

	afPfdTrans.DeleteAllExtAppIDs()
	for appID, pfdData := range pfdMng.PfdDatas {
		afPfdTrans.AddExtAppID(appID)
		pfdDataForApp := convertPfdDataToPfdDataForApp(&pfdData)
		if pfdReport := p.storePfdDataToUDR(appID, pfdDataForApp); pfdReport != nil {
			delete(pfdMng.PfdDatas, appID)
			addPfdReport(pfdMng, pfdReport)
		} else {
			pfdData.Self = genPfdDataURI(p.cfg.GetSbiUri(), scsAsID, afPfdTrans.GetTransID(), appID)
			pfdMng.PfdDatas[appID] = pfdData
			pfdNotifyContext.AddNotification(appID, &models.PfdChangeNotification{
				ApplicationId: appID,
				Pfds:          pfdDataForApp.Pfds,
			})
		}
	}
	if len(pfdMng.PfdDatas) == 0 {
		// The PFDs for all applications were not created successfully.
		// PfdReport is included with detailed information.
		return &HandlerResponse{http.StatusInternalServerError, nil, &pfdMng.PfdReports}
	}

	pfdMng.Self = genPfdManagementURI(p.cfg.GetSbiUri(), scsAsID, afPfdTrans.GetTransID())

	return &HandlerResponse{http.StatusOK, nil, pfdMng}
}

func (p *Processor) DeleteIndividualPFDManagementTransaction(scsAsID, transID string) *HandlerResponse {
	logger.PFDManageLog.Infof("DeleteIndividualPFDManagementTransaction - scsAsID[%s], transID[%s]", scsAsID, transID)

	afCtx, afPfdTrans, err := p.nefCtx.GetAfCtxAndPfdTransWithTransID(scsAsID, transID)
	if err != nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(err.Error())}
	}

	pfdNotifyContext := p.notifier.PfdChangeNotifier.NewPfdNotifyContext()
	defer pfdNotifyContext.FlushNotifications()

	for _, extAppID := range afPfdTrans.GetExtAppIDs() {
		if rsp := p.deletePfdDataFromUDR(extAppID); rsp != nil {
			return rsp
		}
		pfdNotifyContext.AddNotification(extAppID, &models.PfdChangeNotification{
			ApplicationId: extAppID,
			RemovalFlag:   true,
		})
	}
	afCtx.DeletePfdTrans(afPfdTrans.GetTransID())

	// TODO: Remove AfCtx if its subscriptions and transactions are both empty

	return &HandlerResponse{http.StatusNoContent, nil, nil}
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

	afPfdTrans, err := p.nefCtx.GetPfdTransWithAppID(scsAsID, transID, appID)
	if err != nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(err.Error())}
	}

	pfdNotifyContext := p.notifier.PfdChangeNotifier.NewPfdNotifyContext()
	defer pfdNotifyContext.FlushNotifications()

	if rsp := p.deletePfdDataFromUDR(appID); rsp != nil {
		return rsp
	}
	afPfdTrans.DeleteExtAppID(appID)
	pfdNotifyContext.AddNotification(appID, &models.PfdChangeNotification{
		ApplicationId: appID,
		RemovalFlag:   true,
	})

	// TODO: Remove afPfdTrans if its appID is empty

	// TODO: Remove AfCtx if its subscriptions and transactions are both empty

	return &HandlerResponse{http.StatusNoContent, nil, nil}
}

func (p *Processor) PutIndividualApplicationPFDManagement(scsAsID, transID, appID string,
	pfdData *models.PfdData) *HandlerResponse {

	logger.PFDManageLog.Infof("PutIndividualApplicationPFDManagement - scsAsID[%s], transID[%s], appID[%s]",
		scsAsID, transID, appID)

	// TODO: Authorize the AF

	if _, err := p.nefCtx.GetPfdTransWithAppID(scsAsID, transID, appID); err != nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(err.Error())}
	}

	if problemDetails := validatePfdData(pfdData, p.nefCtx, false); problemDetails != nil {
		return &HandlerResponse{int(problemDetails.Status), nil, problemDetails}
	}

	pfdNotifyContext := p.notifier.PfdChangeNotifier.NewPfdNotifyContext()
	defer pfdNotifyContext.FlushNotifications()

	pfdDataForApp := convertPfdDataToPfdDataForApp(pfdData)
	if pfdReport := p.storePfdDataToUDR(appID, pfdDataForApp); pfdReport != nil {
		return &HandlerResponse{http.StatusInternalServerError, nil, pfdReport}
	}
	pfdData.Self = genPfdDataURI(p.cfg.GetSbiUri(), scsAsID, transID, appID)
	pfdNotifyContext.AddNotification(appID, &models.PfdChangeNotification{
		ApplicationId: appID,
		Pfds:          pfdDataForApp.Pfds,
	})

	return &HandlerResponse{http.StatusOK, nil, pfdData}
}

func (p *Processor) PatchIndividualApplicationPFDManagement(scsAsID, transID, appID string,
	pfdData *models.PfdData) *HandlerResponse {

	logger.PFDManageLog.Infof("PatchIndividualApplicationPFDManagement - scsAsID[%s], transID[%s], appID[%s]",
		scsAsID, transID, appID)

	// TODO: Authorize the AF

	if _, err := p.nefCtx.GetPfdTransWithAppID(scsAsID, transID, appID); err != nil {
		return &HandlerResponse{http.StatusNotFound, nil, util.ProblemDetailsDataNotFound(err.Error())}
	}

	if problemDetails := validatePfdData(pfdData, p.nefCtx, true); problemDetails != nil {
		return &HandlerResponse{int(problemDetails.Status), nil, problemDetails}
	}

	pfdNotifyContext := p.notifier.PfdChangeNotifier.NewPfdNotifyContext()
	defer pfdNotifyContext.FlushNotifications()

	rspCode, rspBody := p.consumer.UdrSrv.AppDataPfdsAppIdGet(appID)
	if rspCode != http.StatusOK {
		return &HandlerResponse{rspCode, nil, rspBody}
	}

	oldPfdData := convertPfdDataForAppToPfdData(rspBody.(*models.PfdDataForApp))
	if problemDetails := patchModifyPfdData(oldPfdData, pfdData); problemDetails != nil {
		return &HandlerResponse{int(problemDetails.Status), nil, problemDetails}
	}

	pfdDataForApp := convertPfdDataToPfdDataForApp(oldPfdData)
	if pfdReport := p.storePfdDataToUDR(appID, pfdDataForApp); pfdReport != nil {
		return &HandlerResponse{http.StatusInternalServerError, nil, pfdReport}
	}
	oldPfdData.Self = genPfdDataURI(p.cfg.GetSbiUri(), scsAsID, transID, appID)
	pfdNotifyContext.AddNotification(appID, &models.PfdChangeNotification{
		ApplicationId: appID,
		Pfds:          pfdDataForApp.Pfds,
	})

	return &HandlerResponse{http.StatusOK, nil, oldPfdData}
}

func (p *Processor) buildPfdManagement(afID string, afPfdTrans *context.AfPfdTransaction) (*models.PfdManagement,
	*HandlerResponse) {

	transID := afPfdTrans.GetTransID()
	appIDs := afPfdTrans.GetExtAppIDs()
	pfdMng := &models.PfdManagement{
		Self:     genPfdManagementURI(p.cfg.GetSbiUri(), afID, transID),
		PfdDatas: make(map[string]models.PfdData, len(appIDs)),
	}

	rspCode, rspBody := p.consumer.UdrSrv.AppDataPfdsGet(appIDs)
	if rspCode != http.StatusOK {
		return nil, &HandlerResponse{rspCode, nil, rspBody}
	}
	for _, pfdDataForApp := range *(rspBody.(*[]models.PfdDataForApp)) {
		pfdData := convertPfdDataForAppToPfdData(&pfdDataForApp)
		pfdData.Self = genPfdDataURI(p.cfg.GetSbiUri(), afID, transID, pfdData.ExternalAppId)
		pfdMng.PfdDatas[pfdData.ExternalAppId] = *pfdData
	}
	return pfdMng, nil
}

func (p *Processor) storePfdDataToUDR(appID string, pfdDataForApp *models.PfdDataForApp) *models.PfdReport {
	rspCode, _ := p.consumer.UdrSrv.AppDataPfdsAppIdPut(appID, pfdDataForApp)
	if rspCode != http.StatusCreated && rspCode != http.StatusOK {
		return &models.PfdReport{
			ExternalAppIds: []string{appID},
			FailureCode:    models.FailureCode_MALFUNCTION,
		}
	}
	return nil
}

func (p *Processor) deletePfdDataFromUDR(appID string) *HandlerResponse {
	rspCode, rspBody := p.consumer.UdrSrv.AppDataPfdsAppIdDelete(appID)
	if rspCode != http.StatusNoContent {
		return &HandlerResponse{rspCode, nil, rspBody}
	}
	return nil
}

// The behavior of PATCH update is based on TS 29.250 v1.15.1 clause 4.4.1
func patchModifyPfdData(old, new *models.PfdData) *models.ProblemDetails {
	for pfdID, newPfd := range new.Pfds {
		_, exist := old.Pfds[pfdID]
		if len(newPfd.FlowDescriptions) == 0 && len(newPfd.Urls) == 0 && len(newPfd.DomainNames) == 0 {
			if exist {
				// New Pfd with existing PfdID and empty content implies deletion from old PfdData.
				delete(old.Pfds, pfdID)
			} else {
				// Otherwire, if the PfdID doesn't exist yet, the Pfd still needs valid content.
				return util.ProblemDetailsDataNotFound(DetailNoPfdInfo)
			}
		} else {
			// Either add or update the Pfd to the old PfdData.
			old.Pfds[pfdID] = newPfd
		}
	}
	return nil
}

func convertPfdDataForAppToPfdData(pfdDataForApp *models.PfdDataForApp) *models.PfdData {
	pfdData := &models.PfdData{
		ExternalAppId: pfdDataForApp.ApplicationId,
		Pfds:          make(map[string]models.Pfd, len(pfdDataForApp.Pfds)),
	}
	for _, pfdContent := range pfdDataForApp.Pfds {
		var pfd models.Pfd
		pfd.PfdId = pfdContent.PfdId
		pfd.FlowDescriptions = pfdContent.FlowDescriptions
		pfd.Urls = pfdContent.Urls
		pfd.DomainNames = pfdContent.DomainNames
		pfdData.Pfds[pfdContent.PfdId] = pfd
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
		sbiURI, factory.PfdMngResUriPrefix, afID, transID)
}

func genPfdDataURI(sbiURI, afID, transID, appID string) string {
	// E.g. https://localhost:29505/3gpp-pfd-management/v1/{afID}/transactions/{transID}/applications/{appID}
	return fmt.Sprintf("%s%s/%s/transactions/%s/applications/%s",
		sbiURI, factory.PfdMngResUriPrefix, afID, transID, appID)
}

func validatePfdManagement(afID, transID string, pfdMng *models.PfdManagement,
	nefCtx *context.NefContext) *models.ProblemDetails {

	pfdMng.PfdReports = make(map[string]models.PfdReport)

	if len(pfdMng.PfdDatas) == 0 {
		return util.ProblemDetailsDataNotFound(DetailNoPfdData)
	}

	for appID, pfdData := range pfdMng.PfdDatas {
		// Check whether the received external Application Identifier(s) are already provisioned
		exist, appAfID, appTransID := nefCtx.IsAppIDExisted(appID)
		if exist && (appAfID != afID || appTransID != transID) {
			delete(pfdMng.PfdDatas, appID)
			addPfdReport(pfdMng, &models.PfdReport{
				ExternalAppIds: []string{appID},
				FailureCode:    models.FailureCode_APP_ID_DUPLICATED,
			})
		}
		if problemDetails := validatePfdData(&pfdData, nefCtx, false); problemDetails != nil {
			return problemDetails
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

func validatePfdData(pfdData *models.PfdData, nefCtx *context.NefContext, isPatch bool) *models.ProblemDetails {
	if pfdData.ExternalAppId == "" {
		return util.ProblemDetailsDataNotFound(DetailNoExtAppID)
	}

	if len(pfdData.Pfds) == 0 {
		return util.ProblemDetailsDataNotFound(DetailNoPfd)
	}

	for _, pfd := range pfdData.Pfds {
		if pfd.PfdId == "" {
			return util.ProblemDetailsDataNotFound(DetailNoPfdID)
		}
		// For PATCH method, empty these three attributes is used to imply the deletion of this PFD
		if !isPatch && len(pfd.FlowDescriptions) == 0 && len(pfd.Urls) == 0 && len(pfd.DomainNames) == 0 {
			return util.ProblemDetailsDataNotFound(DetailNoPfdInfo)
		}
	}

	return nil
}

func addPfdReport(pfdMng *models.PfdManagement, newReport *models.PfdReport) {
	if oldReport, ok := pfdMng.PfdReports[string(newReport.FailureCode)]; ok {
		oldReport.ExternalAppIds = append(oldReport.ExternalAppIds, newReport.ExternalAppIds...)
	} else {
		pfdMng.PfdReports[string(newReport.FailureCode)] = *newReport
	}
}
