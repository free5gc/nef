package util

import (
	"net/http"

	"github.com/gin-gonic/gin"

	nef_context "github.com/free5gc/nef/internal/context"
	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi/models"
)

type RouterAuthorizationCheck struct {
	serviceName models.ServiceName
}

func NewRouterAuthorizationCheck(serviceName models.ServiceName) *RouterAuthorizationCheck {
	return &RouterAuthorizationCheck{
		serviceName: serviceName,
	}
}

func (rac *RouterAuthorizationCheck) Check(c *gin.Context, nefContext nef_context.NFContext) {
	token := c.Request.Header.Get("Authorization")
	err := nefContext.AuthorizationCheck(token, rac.serviceName)
	if err != nil {
		logger.CtxLog.Debugf("RouterAuthorizationCheck: Check Unauthorized: %s", err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		c.Abort()
		return
	}

	logger.CtxLog.Debugf("RouterAuthorizationCheck: Check Authorized")
}
