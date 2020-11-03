package consumer

import (
	//ctx "context"
	//"fmt"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/openapi/Npcf_PolicyAuthorization"
	//"bitbucket.org/free5gc-team/openapi/models"
)

type ConsumerPCFService struct {
	cfg              *factory.Config
	nefCtx           *context.NefContext
	clientPolicyAuth *Npcf_PolicyAuthorization.APIClient
}

func NewConsumerPCFService(nefCfg *factory.Config, nefCtx *context.NefContext) *ConsumerPCFService {
	c := &ConsumerPCFService{cfg: nefCfg, nefCtx: nefCtx}

	paConfig := Npcf_PolicyAuthorization.NewConfiguration()
	paConfig.SetBasePath(c.nefCtx.GetPcfURI())
	c.clientPolicyAuth = Npcf_PolicyAuthorization.NewAPIClient(paConfig)
	return c
}

func (c *ConsumerPCFService) PostAppSessions() error {
	//asc := models.AppSessionContext{}
	//result, rsp, err := c.clientPolicyAuth.ApplicationSessionsCollectionApi.PostAppSessions(ctx.Background(), asc)
	//if err != nil {
	//	return fmt.Errorf("PostAppSessions Error: %+v", err)
	//}
	return nil
}
