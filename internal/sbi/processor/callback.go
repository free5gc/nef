package processor

import (
	"net/http"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/models"
)

func (p *Processor) SmfNotification(
	eeNotif *models.NsmfEventExposureNotification,
) *HandlerResponse {
	logger.TrafInfluLog.Infof("SmfNotification - NotifId[%s]", eeNotif.NotifId)

	af, sub := p.Context().FindAfSub(eeNotif.NotifId)
	if sub == nil {
		pd := openapi.ProblemDetailsDataNotFound("Subscrption is not found")
		return &HandlerResponse{http.StatusNotFound, nil, pd}
	}

	af.Mu.RLock()
	defer af.Mu.RUnlock()

	// TODO: Notify AF

	return &HandlerResponse{http.StatusOK, nil, nil}
}
