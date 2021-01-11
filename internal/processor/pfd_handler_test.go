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
	"bitbucket.org/free5gc-team/nef/internal/util"
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
	pfd3 = models.Pfd{
		PfdId: "pfd3",
		Urls: []string{
			"^http://test.example2.net(/\\S*)?$",
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

func TestGetIndividualPFDManagementTransaction(t *testing.T) {
	initUDRDrGetPfdDatasStub()
	defer gock.Off()

	testCases := []struct {
		name             string
		afID             string
		transID          string
		expectedResponse *HandlerResponse
	}{
		{
			name:    "Valid input",
			afID:    "af1",
			transID: "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body: &models.PfdManagement{
					Self: genPfdManagementURI(nefProcessor.cfg.GetSbiUri(), "af1", "1"),
					PfdDatas: map[string]models.PfdData{
						"app1": {
							ExternalAppId: "app1",
							Self:          genPfdDataURI(nefProcessor.cfg.GetSbiUri(), "af1", "1", "app1"),
							Pfds: map[string]models.Pfd{
								"pfd1": pfd1,
								"pfd2": pfd2,
							},
						},
						"app2": {
							ExternalAppId: "app2",
							Self:          genPfdDataURI(nefProcessor.cfg.GetSbiUri(), "af1", "1", "app2"),
							Pfds: map[string]models.Pfd{
								"pfd3": pfd3,
							},
						},
					},
				},
			},
		},
		{
			name:    "Invalid ID test",
			afID:    "af1",
			transID: "-1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   util.ProblemDetailsDataNotFound("Transaction not found"),
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
			afPfdTans.AddExtAppID("app2")

			rsp := nefProcessor.GetIndividualPFDManagementTransaction(tc.afID, tc.transID)
			validateResult(t, tc.expectedResponse, rsp)
		})
	}
}

func TestDeleteIndividualPFDManagementTransaction(t *testing.T) {
	initUDRDrDeletePfdDataStub()
	defer gock.Off()

	testCases := []struct {
		name             string
		afID             string
		transID          string
		expectedResponse *HandlerResponse
	}{
		{
			name:    "Valid input",
			afID:    "af1",
			transID: "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNoContent,
			},
		},
		{
			name:    "Invalid ID test",
			afID:    "af2",
			transID: "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   util.ProblemDetailsDataNotFound("AF not found"),
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

			rsp := nefProcessor.DeleteIndividualPFDManagementTransaction(tc.afID, tc.transID)
			validateResult(t, tc.expectedResponse, rsp)
		})
	}
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

func TestPutIndividualApplicationPFDManagement(t *testing.T) {
	initUDRDrPutPfdDataStub(http.StatusOK)
	defer gock.Off()

	testCases := []struct {
		name             string
		afID             string
		transID          string
		appID            string
		pfdData          *models.PfdData
		expectedResponse *HandlerResponse
	}{
		{
			name:    "Valid input",
			afID:    "af1",
			transID: "1",
			appID:   "app1",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
					"pfd2": pfd2,
				},
			},
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
			pfdData: &models.PfdData{
				ExternalAppId: "app2",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
					"pfd2": pfd2,
				},
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   util.ProblemDetailsDataNotFound("Application ID not found"),
			},
		},
		{
			name:    "Invalid PfdData test",
			afID:    "af1",
			transID: "1",
			appID:   "app1",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": {
						PfdId: "pfd1",
					},
				},
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   util.ProblemDetailsDataNotFound(PFD_ERR_NO_FLOW_IDENT),
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

			rsp := nefProcessor.PutIndividualApplicationPFDManagement(tc.afID, tc.transID, tc.appID, tc.pfdData)
			validateResult(t, tc.expectedResponse, rsp)
		})
	}
}

func TestPatchIndividualApplicationPFDManagement(t *testing.T) {
	initUDRDrGetPfdDataStub()
	initUDRDrPutPfdDataStub(http.StatusOK)
	defer gock.Off()

	testCases := []struct {
		name             string
		afID             string
		transID          string
		appID            string
		pfdData          *models.PfdData
		expectedResponse *HandlerResponse
	}{
		{
			name:    "Valid input",
			afID:    "af1",
			transID: "1",
			appID:   "app1",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": {
						PfdId: "pfd1",
					},
				},
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body: &models.PfdData{
					ExternalAppId: "app1",
					Self:          genPfdDataURI(nefProcessor.cfg.GetSbiUri(), "af1", "1", "app1"),
					Pfds: map[string]models.Pfd{
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
			pfdData: &models.PfdData{
				ExternalAppId: "app2",
				Pfds: map[string]models.Pfd{
					"pfd1": {
						PfdId: "pfd1",
					},
				},
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   util.ProblemDetailsDataNotFound("Application ID not found"),
			},
		},
		{
			name:    "Invalid PfdData test",
			afID:    "af1",
			transID: "1",
			appID:   "app1",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd3": {
						PfdId: "pfd3",
					},
				},
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   util.ProblemDetailsDataNotFound(PFD_ERR_NO_FLOW_IDENT),
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

			rsp := nefProcessor.PatchIndividualApplicationPFDManagement(tc.afID, tc.transID, tc.appID, tc.pfdData)
			validateResult(t, tc.expectedResponse, rsp)
		})
	}
}

func TestValidatePfdManagement(t *testing.T) {
	testCases := []struct {
		name            string
		pfdManagement   *models.PfdManagement
		expectedProblem *models.ProblemDetails
		expectedReports map[string]models.PfdReport
	}{
		{
			name: "Valid",
			pfdManagement: &models.PfdManagement{
				PfdDatas: map[string]models.PfdData{
					"app1": {
						ExternalAppId: "app1",
						Pfds: map[string]models.Pfd{
							"pfd1": pfd1,
							"pfd2": pfd2,
						},
					},
					"app2": {
						ExternalAppId: "app2",
						Pfds: map[string]models.Pfd{
							"pfd3": pfd3,
						},
					},
				},
			},
			expectedProblem: nil,
			expectedReports: map[string]models.PfdReport{},
		},
		{
			name: "Invalid, empty PfdDatas",
			pfdManagement: &models.PfdManagement{
				PfdDatas: map[string]models.PfdData{},
			},
			expectedProblem: util.ProblemDetailsDataNotFound(PFD_ERR_NO_PFD_DATA),
			expectedReports: map[string]models.PfdReport{},
		},
		{
			name: "Invalid, an appID is already provisioned",
			pfdManagement: &models.PfdManagement{
				PfdDatas: map[string]models.PfdData{
					"app100": {
						ExternalAppId: "app100",
						Pfds: map[string]models.Pfd{
							"pfd1": pfd1,
						},
					},
					"app101": {
						ExternalAppId: "app101",
						Pfds: map[string]models.Pfd{
							"pfd1": pfd1,
						},
					},
				},
			},
			expectedProblem: nil,
			expectedReports: map[string]models.PfdReport{
				string(models.FailureCode_APP_ID_DUPLICATED): {
					ExternalAppIds: []string{"app100"},
					FailureCode:    models.FailureCode_APP_ID_DUPLICATED,
				},
			},
		},
		{
			name: "Invalid, none of the PFDs were created",
			pfdManagement: &models.PfdManagement{
				PfdDatas: map[string]models.PfdData{
					"app100": {
						ExternalAppId: "app100",
						Pfds: map[string]models.Pfd{
							"pfd1": pfd1,
						},
					},
				},
			},
			expectedProblem: util.ProblemDetailsSystemFailure("None of the PFDs were created"),
			expectedReports: map[string]models.PfdReport{
				string(models.FailureCode_APP_ID_DUPLICATED): {
					ExternalAppIds: []string{"app100"},
					FailureCode:    models.FailureCode_APP_ID_DUPLICATED,
				},
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
			afPfdTans.AddExtAppID("app100")

			rst := validatePfdManagement(tc.pfdManagement, nefContext)
			validateResult(t, tc.expectedProblem, rst)
			validateResult(t, tc.expectedReports, tc.pfdManagement.PfdReports)
		})
	}
}

func TestValidatePfdData(t *testing.T) {
	testCases := []struct {
		name           string
		pfdData        *models.PfdData
		expectedResult *models.ProblemDetails
	}{
		{
			name: "Valid",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
				},
			},
			expectedResult: nil,
		},
		{
			name: "Invalid, without ExternalAppId",
			pfdData: &models.PfdData{
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
				},
			},
			expectedResult: util.ProblemDetailsDataNotFound(PFD_ERR_NO_EXTERNAL_APP_ID),
		},
		{
			name: "Invalid, empty Pfds",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
			},
			expectedResult: util.ProblemDetailsDataNotFound(PFD_ERR_NO_PFD),
		},
		{
			name: "Invalid, without PfdID",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": {
						FlowDescriptions: []string{
							"permit in ip from 10.68.28.39 80 to any",
							"permit out ip from any to 10.68.28.39 80",
						},
					},
				},
			},
			expectedResult: util.ProblemDetailsDataNotFound(PFD_ERR_NO_PFD_ID),
		},
		{
			name: "Invalid, FlowDescriptions, Urls and DomainNames are all empty",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": {
						PfdId: "pfd1",
					},
				},
			},
			expectedResult: util.ProblemDetailsDataNotFound(PFD_ERR_NO_FLOW_IDENT),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rst := validatePfdData(tc.pfdData, nefContext, false)
			validateResult(t, tc.expectedResult, rst)
		})
	}
}

func TestPatchModifyPfdData(t *testing.T) {
	testCases := []struct {
		name            string
		old             *models.PfdData
		new             *models.PfdData
		expectedProblem *models.ProblemDetails
		expectedResult  *models.PfdData
	}{
		{
			name: "Add Pfd",
			old: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
				},
			},
			new: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd2": pfd2,
				},
			},
			expectedProblem: nil,
			expectedResult: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
					"pfd2": pfd2,
				},
			},
		},
		{
			name: "Update Pfd",
			old: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
				},
			},
			new: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": {
						PfdId: "pfd1",
						Urls: []string{
							"^http://test.example.com(/\\S*)?$",
						},
					},
				},
			},
			expectedProblem: nil,
			expectedResult: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": {
						PfdId: "pfd1",
						Urls: []string{
							"^http://test.example.com(/\\S*)?$",
						},
					},
				},
			},
		},
		{
			name: "Delete Pfd",
			old: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
					"pfd2": pfd2,
				},
			},
			new: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": {
						PfdId: "pfd1",
					},
				},
			},
			expectedProblem: nil,
			expectedResult: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd2": pfd2,
				},
			},
		},
		{
			name: "Invalid Update Pfd",
			old: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
				},
			},
			new: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd2": {
						PfdId: "pfd2",
					},
				},
			},
			expectedProblem: util.ProblemDetailsDataNotFound(PFD_ERR_NO_FLOW_IDENT),
			expectedResult: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			problemDetail := patchModifyPfdData(tc.old, tc.new)
			validateResult(t, tc.expectedProblem, problemDetail)
			validateResult(t, tc.expectedResult, tc.old)
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

func initUDRDrGetPfdDatasStub() {
	pfdDataForApp := []models.PfdDataForApp{
		{
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
		},
		{
			ApplicationId: "app2",
			Pfds: []models.PfdContent{
				{
					PfdId: "pfd3",
					Urls: []string{
						"^http://test.example2.net(/\\S*)?$",
					},
				},
			},
		},
	}

	gock.New("http://127.0.0.4:8000/nudr-dr/v1").
		Get("/application-data/pfds").
		ParamPresent("appId").
		Persist().
		Reply(http.StatusOK).
		JSON(pfdDataForApp)
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

func initUDRDrPutPfdDataStub(statusCode int) {
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
		Put("/application-data/pfds/.*").
		Persist().
		Reply(statusCode).
		JSON(pfdDataForApp)
}
