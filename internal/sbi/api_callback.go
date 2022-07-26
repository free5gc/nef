package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"bitbucket.org/free5gc-team/openapi/models"
)

func (s *Server) getCallbackEndpoints() []Endpoint {
	return []Endpoint{
		{
			Method:  http.MethodPost,
			Pattern: "/notification/smf",
			APIFunc: s.apiPostSmfNotification,
		},
	}
}

func (s *Server) apiPostSmfNotification(gc *gin.Context) {
	contentType, err := checkContentTypeIsJSON(gc)
	if err != nil {
		return
	}

	var eeNotif models.NsmfEventExposureNotification
	if err := s.deserializeData(gc, &eeNotif, contentType); err != nil {
		return
	}

	hdlRsp := s.Processor().SmfNotification(&eeNotif)

	s.buildAndSendHttpResponse(gc, hdlRsp, false)
}
