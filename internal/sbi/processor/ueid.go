package processor

import (
	"net/http"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/gin-gonic/gin"
)

func (p *Processor) TranslateUeId(
	c *gin.Context,
	afID string,
	req *models.Nef_UEId_UeIdTranslationReqData,
) {
	if req.Gpsi == "" {
		pd := openapi.ProblemDetailsMalformedReqSyntax("gpsi is required")
		c.JSON(http.StatusBadRequest, pd)
		return
	}

	if p.Context().GetAf(afID) == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF not found: " + afID)
		c.JSON(http.StatusNotFound, pd)
		return
	}

	supi, pd, err := p.Consumer().GetSupiFromGpsi(req.Gpsi)
	if err != nil {
		logger.SBILog.Errorf("TranslateUeId GetSupiFromGpsi err: %+v", err)
		c.JSON(http.StatusInternalServerError, openapi.ProblemDetailsSystemFailure(err.Error()))
		return
	}
	if pd != nil {
		c.JSON(int(pd.Status), pd)
		return
	}

	c.JSON(http.StatusOK, &models.Nef_UEId_UeIdTranslationRspData{
		Supi: supi,
		Gpsi: req.Gpsi,
	})
}
