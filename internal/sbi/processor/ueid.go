package processor

import (
	"net/http"
	"strings"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/gin-gonic/gin"
)

// RetrieveUeId retrieves the AF-specific external identifier of a UE identified by its
// IP or MAC address, per 3GPP TS 29.522 clause 4.4.21 (3gpp-ueid / Nnef_UEIdentifierTranslation).
//
// NEF resolves the UE's SUPI internally via BSF's Nbsf_Management_Discovery (keyed by the
// UE's IP/MAC address), then asks UDM (Nudm_SDM) for the external ID valid for the
// requesting AF. The SUPI itself is never returned to the AF: TS 33.501 clause 5.9.2.3
// prohibits NEF from sending SUPI outside the 3GPP operator domain.
func (p *Processor) RetrieveUeId(
	c *gin.Context,
	req *models.Nef_UEId_UeIdReq,
) {
	problemDetails := validateUeIdReq(req)
	if problemDetails != nil {
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	var ipv4Addr, macAddr48 string
	if req.UeMacAddr != "" {
		macAddr48 = req.UeMacAddr
	} else {
		ipv4Addr = req.UeIpAddr.Ipv4Addr
	}

	supi, pd, err := p.Consumer().GetSupiFromUeAddr(ipv4Addr, macAddr48, req.Dnn)
	switch {
	case pd != nil:
		c.JSON(int(pd.Status), pd)
		return
	case err != nil:
		pd := openapi.ProblemDetailsSystemFailure("Query to BSF failed")
		c.JSON(int(pd.Status), pd)
		return
	case supi == "":
		pd := openapi.ProblemDetailsDataNotFound("No PCF binding found for the given UE address")
		c.JSON(int(pd.Status), pd)
		return
	}

	gpsi, pd, err := p.Consumer().GetGpsiFromSupi(supi, req.AfId, req.MtcProviderId)
	switch {
	case pd != nil:
		c.JSON(int(pd.Status), pd)
		return
	case err != nil:
		pd := openapi.ProblemDetailsSystemFailure("Query to UDM failed")
		c.JSON(int(pd.Status), pd)
		return
	}

	externalId, ok := strings.CutPrefix(gpsi, "extid-")
	if !ok {
		// This UE has only an MSISDN-formatted Gpsi, not an External Identifier; UeIdInfo
		// has no field to carry an MSISDN, so there is no externalId to return.
		pd := openapi.ProblemDetailsDataNotFound("No external identifier found for the given UE")
		c.JSON(int(pd.Status), pd)
		return
	}

	c.JSON(http.StatusOK, &models.Nef_UEId_UeIdInfo{ExternalId: externalId})
}

func validateUeIdReq(req *models.Nef_UEId_UeIdReq) *models.ProblemDetails {
	if req.AfId == "" {
		return openapi.ProblemDetailsMalformedReqSyntax("Missing afId")
	}

	ip := req.UeIpAddr
	if ip != nil && ip.Ipv4Addr == "" && (ip.Ipv6Addr != "" || ip.Ipv6Prefix != "") {
		// v1 only supports IPv4 lookups against BSF; IPv6-keyed lookups are not yet wired up.
		return openapi.ProblemDetailsMalformedReqSyntax(
			"IPv6-keyed ueIpAddr is not yet supported; provide ueIpAddr.ipv4Addr or ueMacAddr")
	}

	haveIp := req.UeIpAddr != nil && req.UeIpAddr.Ipv4Addr != ""
	haveMac := req.UeMacAddr != ""
	if haveIp == haveMac {
		return openapi.ProblemDetailsMalformedReqSyntax("Exactly one of ueIpAddr, ueMacAddr must be present")
	}

	return nil
}
