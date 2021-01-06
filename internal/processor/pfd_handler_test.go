package processor

import (
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"testing"

	"gopkg.in/h2non/gock.v1"

	"bitbucket.org/free5gc-team/nef/internal/consumer"
	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/models"
)

var (
	nefContext   *context.NefContext
	nefProcessor *Processor

	pfd1 = models.Pfd{
		PfdId: "pfd1",
		FlowDescriptions: []string{
			"permit in ip from 10.68.28.39 80 to any",
			"permit out ip from any to 10.68.28.39 80",
		},
	}
	pfd2 = models.Pfd{
		PfdId: "pfd2",
		Urls: []string{
			"^http://test.example.com(/\\S*)?$",
		},
	}
)

func TestMain(m *testing.M) {
	openapi.InterceptH2CClient()
	initNRFNfmStub()
	initNRFDiscStub()

	nefConfig := &factory.Config{}
	if err := factory.InitConfigFactory("", nefConfig); err != nil {
		return
	}
	nefContext = context.NewNefContext()
	nefConsumer := consumer.NewConsumer(nefConfig, nefContext)
	nefProcessor = NewProcessor(nefConfig, nefContext, nefConsumer)

	exitVal := m.Run()
	openapi.RestoreH2CClient()
	os.Exit(exitVal)
}

func TestGetIndividualApplicationPFDManagement(t *testing.T) {
	initUDRDrGetPfdDataStub()
	defer gock.Off()

	testCases := []struct {
		name             string
		afID             string
		transID          string
		appID            string
		expectedResponse *HandlerResponse
	}{
		{
			name:    "Valid input",
			afID:    "af1",
			transID: "1",
			appID:   "app1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body: &models.PfdData{
					ExternalAppId: "app1",
					Self:          genPfdDataURI(nefProcessor.cfg.GetSbiUri(), "af1", "1", "app1"),
					Pfds: map[string]models.Pfd{
						"pfd1": pfd1,
						"pfd2": pfd2,
					},
				},
			},
		},
		{
			name:    "Invalid ID test",
			afID:    "af1",
			transID: "1",
			appID:   "app2",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   util.ProblemDetailsDataNotFound("Application ID not found"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			afCtx := nefContext.NewAfCtx("af1")
			nefContext.AddAfCtx(afCtx)
			defer nefContext.DeleteAfCtx("af1")
			afPfdTans := nefContext.NewAfPfdTrans(afCtx)
			afCtx.AddPfdTrans(afPfdTans)
			afPfdTans.AddExtAppID("app1")

			rsp := nefProcessor.GetIndividualApplicationPFDManagement(tc.afID, tc.transID, tc.appID)
			validateResult(t, tc.expectedResponse, rsp)
		})
	}
}

func TestDeleteIndividualApplicationPFDManagement(t *testing.T) {
	initUDRDrDeletePfdDataStub()
	defer gock.Off()

	testCases := []struct {
		name             string
		afID             string
		transID          string
		appID            string
		expectedResponse *HandlerResponse
	}{
		{
			name:    "Valid input",
			afID:    "af1",
			transID: "1",
			appID:   "app1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNoContent,
			},
		},
		{
			name:    "Invalid ID test",
			afID:    "af1",
			transID: "1",
			appID:   "app2",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   util.ProblemDetailsDataNotFound("Application ID not found"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			afCtx := nefContext.NewAfCtx("af1")
			nefContext.AddAfCtx(afCtx)
			defer nefContext.DeleteAfCtx("af1")
			afPfdTans := nefContext.NewAfPfdTrans(afCtx)
			afCtx.AddPfdTrans(afPfdTans)
			afPfdTans.AddExtAppID("app1")

			rsp := nefProcessor.DeleteIndividualApplicationPFDManagement(tc.afID, tc.transID, tc.appID)
			validateResult(t, tc.expectedResponse, rsp)
		})
	}
}

func validateResult(t *testing.T, expected, got interface{}) {
	if !reflect.DeepEqual(expected, got) {
		e, err := json.MarshalIndent(expected, "", "  ")
		if err != nil {
			t.Error(err)
		}
		g, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Error(err)
		}
		t.Errorf("Expected response:\n%v\ngot:\n%v\n", string(e), string(g))
	}
}

func initNRFNfmStub() {
	nrfRegisterInstanceRsp := models.NfProfile{
		NfInstanceId: "nef-pfd-unit-testing",
	}
	gock.New("http://127.0.0.10:8000/nnrf-nfm/v1").
		Put("/nf-instances/.*").
		MatchType("json").
		JSON(".*").
		Reply(http.StatusCreated).
		SetHeader("Location", "http://127.0.0.10:8000/nnrf-nfm/v1/nf-instances/12345").
		JSON(nrfRegisterInstanceRsp)
}

func initNRFDiscStub() {
	searchResult := &models.SearchResult{
		ValidityPeriod: 100,
		NfInstances: []models.NfProfile{
			{
				NfInstanceId: "udr-unit-testing",
				NfType:       "UDR",
				NfStatus:     "REGISTERED",
				UdrInfo: &models.UdrInfo{
					SupportedDataSets: []models.DataSetId{
						"SUBSCRIPTION",
					},
				},
				NfServices: &[]models.NfService{
					{
						ServiceInstanceId: "datarepository",
						ServiceName:       "nudr-dr",
						Versions: &[]models.NfServiceVersion{
							{
								ApiVersionInUri: "v1",
								ApiFullVersion:  "1.0.0",
							},
						},
						Scheme:          "http",
						NfServiceStatus: "REGISTERED",
						IpEndPoints: &[]models.IpEndPoint{
							{
								Ipv4Address: "127.0.0.4",
								Transport:   "TCP",
								Port:        8000,
							},
						},
						ApiPrefix: "http://127.0.0.4:8000",
					},
				},
			},
		},
	}

	gock.New("http://127.0.0.10:8000/nnrf-disc/v1").
		Get("/nf-instances").
		MatchParam("target-nf-type", "UDR").
		MatchParam("requester-nf-type", "NEF").
		MatchParam("service-names", "nudr-dr").
		Reply(http.StatusOK).
		JSON(searchResult)
}

func initUDRDrGetPfdDataStub() {
	pfdDataForApp := models.PfdDataForApp{
		ApplicationId: "app1",
		Pfds: []models.PfdContent{
			{
				PfdId: "pfd1",
				FlowDescriptions: []string{
					"permit in ip from 10.68.28.39 80 to any",
					"permit out ip from any to 10.68.28.39 80",
				},
			},
			{
				PfdId: "pfd2",
				Urls: []string{
					"^http://test.example.com(/\\S*)?$",
				},
			},
		},
	}

	gock.New("http://127.0.0.4:8000/nudr-dr/v1").
		Get("/application-data/pfds/.*").
		Persist().
		Reply(http.StatusOK).
		JSON(pfdDataForApp)
}

func initUDRDrDeletePfdDataStub() {
	gock.New("http://127.0.0.4:8000/nudr-dr/v1").
		Delete("/application-data/pfds/.*").
		Persist().
		Reply(http.StatusNoContent)
}
