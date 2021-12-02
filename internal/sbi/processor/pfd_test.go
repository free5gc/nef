package processor

import (
	"net/http"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/h2non/gock.v1"

	nefctx "bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/sbi/consumer"
	"bitbucket.org/free5gc-team/nef/internal/sbi/notifier"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/models"
)

type nefTestApp struct {
	cfg      *factory.Config
	nefCtx   *nefctx.NefContext
	consumer *consumer.Consumer
	notifier *notifier.Notifier
	proc     *Processor
}

func newTestApp(cfg *factory.Config, tlsKeyLogPath string) (*nefTestApp, error) {
	var err error
	nef := &nefTestApp{cfg: cfg}

	if nef.nefCtx, err = nefctx.NewContext(nef); err != nil {
		return nil, err
	}
	if nef.consumer, err = consumer.NewConsumer(nef); err != nil {
		return nil, err
	}
	if nef.notifier, err = notifier.NewNotifier(); err != nil {
		return nil, err
	}
	if nef.proc, err = NewProcessor(nef); err != nil {
		return nil, err
	}
	return nef, nil
}

func (a *nefTestApp) Config() *factory.Config {
	return a.cfg
}

func (a *nefTestApp) Context() *nefctx.NefContext {
	return a.nefCtx
}

func (a *nefTestApp) Consumer() *consumer.Consumer {
	return a.consumer
}

func (a *nefTestApp) Notifier() *notifier.Notifier {
	return a.notifier
}

func (a *nefTestApp) Processor() *Processor {
	return a.proc
}

var (
	nefApp *nefTestApp

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

	pfdDataForApp1 = models.PfdDataForApp{
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
	pfdDataForApp2 = models.PfdDataForApp{
		ApplicationId: "app2",
		Pfds: []models.PfdContent{
			{
				PfdId: "pfd3",
				Urls: []string{
					"^http://test.example2.net(/\\S*)?$",
				},
			},
		},
	}
)

func TestMain(m *testing.M) {
	var err error
	openapi.InterceptH2CClient()
	initNRFNfmStub()
	initNRFDiscStub()

	cfg := &factory.Config{
		Info: &factory.Info{
			Version: "1.0.0",
		},
		Configuration: &factory.Configuration{
			Sbi: &factory.Sbi{
				Scheme:       "http",
				RegisterIPv4: "127.0.0.5",
				BindingIPv4:  "127.0.0.5",
				Port:         8000,
			},
			NrfUri: "http://127.0.0.10:8000",
			ServiceList: []factory.Service{
				{
					ServiceName: "nnef-pfdmanagement",
				},
			},
		},
	}
	nefApp, err = newTestApp(cfg, "")
	if err != nil {
		panic(err)
	}

	exitVal := m.Run()
	openapi.RestoreH2CClient()
	os.Exit(exitVal)
}

func TestGetPFDManagementTransactions(t *testing.T) {
	initUDRDrGetPfdDatasStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body: &[]models.PfdManagement{
					{
						Self: genPfdManagementURI(nefApp.Config().SbiUri(), "af1", "1"),
						PfdDatas: map[string]models.PfdData{
							"app1": {
								ExternalAppId: "app1",
								Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app1"),
								Pfds: map[string]models.Pfd{
									"pfd1": pfd1,
									"pfd2": pfd2,
								},
							},
							"app2": {
								ExternalAppId: "app2",
								Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app2"),
								Pfds: map[string]models.Pfd{
									"pfd3": pfd3,
								},
							},
						},
					},
				},
			},
		},
		{
			description: "Invalid AF ID, should return ProblemDetails",
			afID:        "af2",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound(DetailNoAF),
			},
		},
	}

	Convey("Given AF ID, should return PfdManagements belonging to this AF", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)
				afPfdTans.AddExtAppID("app1")
				afPfdTans.AddExtAppID("app2")

				rsp := nefApp.Processor().GetPFDManagementTransactions(tc.afID)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestDeletePFDManagementTransactions(t *testing.T) {
	initUDRDrDeletePfdDataStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNoContent,
			},
		},
		{
			description: "Invalid AF ID, should return ProblemDetails",
			afID:        "af2",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound(DetailNoAF),
			},
		},
	}

	Convey("Given AF ID, should delete PfdManagements belonging to this AF", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)

				rsp := nefApp.Processor().DeletePFDManagementTransactions(tc.afID)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestPostPFDManagementTransactions(t *testing.T) {
	initUDRDrPutPfdDataStub(http.StatusCreated)
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		pfdManagement    *models.PfdManagement
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
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
			expectedResponse: &HandlerResponse{
				Status: http.StatusCreated,
				Body: &models.PfdManagement{
					Self: genPfdManagementURI(nefApp.Config().SbiUri(), "af1", "1"),
					PfdDatas: map[string]models.PfdData{
						"app1": {
							ExternalAppId: "app1",
							Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app1"),
							Pfds: map[string]models.Pfd{
								"pfd1": pfd1,
								"pfd2": pfd2,
							},
						},
						"app2": {
							ExternalAppId: "app2",
							Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app2"),
							Pfds: map[string]models.Pfd{
								"pfd3": pfd3,
							},
						},
					},
					PfdReports: map[string]models.PfdReport{},
				},
			},
		},
		{
			description: "Invalid AF ID, should return ProblemDetails",
			afID:        "af2",
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
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound(DetailNoAF),
			},
		},
		{
			description: "Invalid PfdManagement, should return ProblemDetails",
			afID:        "af1",
			pfdManagement: &models.PfdManagement{
				PfdDatas: map[string]models.PfdData{},
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound(DetailNoPfdData),
			},
		},
	}

	Convey("Given AF ID, should add a PfdManagement belonging to this AF", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")

				rsp := nefApp.Processor().PostPFDManagementTransactions(tc.afID, tc.pfdManagement)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestGetIndividualPFDManagementTransaction(t *testing.T) {
	initUDRDrGetPfdDatasStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		transID          string
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
			transID:     "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body: &models.PfdManagement{
					Self: genPfdManagementURI(nefApp.Config().SbiUri(), "af1", "1"),
					PfdDatas: map[string]models.PfdData{
						"app1": {
							ExternalAppId: "app1",
							Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app1"),
							Pfds: map[string]models.Pfd{
								"pfd1": pfd1,
								"pfd2": pfd2,
							},
						},
						"app2": {
							ExternalAppId: "app2",
							Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app2"),
							Pfds: map[string]models.Pfd{
								"pfd3": pfd3,
							},
						},
					},
				},
			},
		},
		{
			description: "Invalid transaction ID, should return ProblemDetails",
			afID:        "af1",
			transID:     "-1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound("Transaction not found"),
			},
		},
	}

	Convey("Given AF and transaction ID, should return the PfdManagement", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)
				afPfdTans.AddExtAppID("app1")
				afPfdTans.AddExtAppID("app2")

				rsp := nefApp.Processor().GetIndividualPFDManagementTransaction(tc.afID, tc.transID)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestDeleteIndividualPFDManagementTransaction(t *testing.T) {
	initUDRDrDeletePfdDataStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		transID          string
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
			transID:     "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNoContent,
			},
		},
		{
			description: "Invalid transaction ID, should return ProblemDetails",
			afID:        "af2",
			transID:     "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound("AF not found"),
			},
		},
	}

	Convey("Given AF and transaction ID, should delete the PfdManagement", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)

				rsp := nefApp.Processor().DeleteIndividualPFDManagementTransaction(tc.afID, tc.transID)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestPutIndividualPFDManagementTransaction(t *testing.T) {
	initUDRDrPutPfdDataStub(http.StatusOK)
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		transID          string
		pfdManagement    *models.PfdManagement
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
			transID:     "1",
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
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body: &models.PfdManagement{
					Self: genPfdManagementURI(nefApp.Config().SbiUri(), "af1", "1"),
					PfdDatas: map[string]models.PfdData{
						"app1": {
							ExternalAppId: "app1",
							Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app1"),
							Pfds: map[string]models.Pfd{
								"pfd1": pfd1,
								"pfd2": pfd2,
							},
						},
						"app2": {
							ExternalAppId: "app2",
							Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app2"),
							Pfds: map[string]models.Pfd{
								"pfd3": pfd3,
							},
						},
					},
					PfdReports: map[string]models.PfdReport{},
				},
			},
		},
		{
			description: "Invalid transaction ID, should return ProblemDetails",
			afID:        "af1",
			transID:     "-1",
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
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound("Transaction not found"),
			},
		},
		{
			description: "Invalid PfdManagement, should return ProblemDetails",
			afID:        "af1",
			transID:     "1",
			pfdManagement: &models.PfdManagement{
				PfdDatas: map[string]models.PfdData{},
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound(DetailNoPfdData),
			},
		},
	}

	Convey("Given AF and transaction ID, should update the PfdManagement", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)

				rsp := nefApp.Processor().PutIndividualPFDManagementTransaction(tc.afID, tc.transID, tc.pfdManagement)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestGetIndividualApplicationPFDManagement(t *testing.T) {
	initUDRDrGetPfdDataStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		transID          string
		appID            string
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
			transID:     "1",
			appID:       "app1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body: &models.PfdData{
					ExternalAppId: "app1",
					Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app1"),
					Pfds: map[string]models.Pfd{
						"pfd1": pfd1,
						"pfd2": pfd2,
					},
				},
			},
		},
		{
			description: "Invalid App ID, should return ProblemDetails",
			afID:        "af1",
			transID:     "1",
			appID:       "app2",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound("Application ID not found"),
			},
		},
	}

	Convey("Given AF, transaction and App ID, should delete the PfdData", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)
				afPfdTans.AddExtAppID("app1")

				rsp := nefApp.Processor().GetIndividualApplicationPFDManagement(tc.afID, tc.transID, tc.appID)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestDeleteIndividualApplicationPFDManagement(t *testing.T) {
	initUDRDrDeletePfdDataStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		transID          string
		appID            string
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
			transID:     "1",
			appID:       "app1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNoContent,
			},
		},
		{
			description: "Invalid App ID, should return ProblemDetails",
			afID:        "af1",
			transID:     "1",
			appID:       "app2",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound("Application ID not found"),
			},
		},
	}

	Convey("Given AF, transaction and App ID, should delete the PfdData", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)
				afPfdTans.AddExtAppID("app1")

				rsp := nefApp.Processor().DeleteIndividualApplicationPFDManagement(tc.afID, tc.transID, tc.appID)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestPutIndividualApplicationPFDManagement(t *testing.T) {
	initUDRDrPutPfdDataStub(http.StatusOK)
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		transID          string
		appID            string
		pfdData          *models.PfdData
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
			transID:     "1",
			appID:       "app1",
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
					Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app1"),
					Pfds: map[string]models.Pfd{
						"pfd1": pfd1,
						"pfd2": pfd2,
					},
				},
			},
		},
		{
			description: "Invalid App ID, should return ProblemDetails",
			afID:        "af1",
			transID:     "1",
			appID:       "app2",
			pfdData: &models.PfdData{
				ExternalAppId: "app2",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
					"pfd2": pfd2,
				},
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   openapi.ProblemDetailsDataNotFound("Application ID not found"),
			},
		},
		{
			description: "Invalid PfdData, should return ProblemDetails",
			afID:        "af1",
			transID:     "1",
			appID:       "app1",
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
				Body:   openapi.ProblemDetailsDataNotFound(DetailNoPfdInfo),
			},
		},
	}

	Convey("Given AF, transaction and App ID, should update the PfdData", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)
				afPfdTans.AddExtAppID("app1")

				rsp := nefApp.Processor().PutIndividualApplicationPFDManagement(tc.afID, tc.transID, tc.appID, tc.pfdData)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestPatchIndividualApplicationPFDManagement(t *testing.T) {
	initUDRDrGetPfdDataStub()
	initUDRDrPutPfdDataStub(http.StatusOK)
	defer gock.Off()

	testCases := []struct {
		description      string
		afID             string
		transID          string
		appID            string
		pfdData          *models.PfdData
		expectedResponse *HandlerResponse
	}{
		{
			description: "Valid input",
			afID:        "af1",
			transID:     "1",
			appID:       "app1",
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
					Self:          genPfdDataURI(nefApp.Config().SbiUri(), "af1", "1", "app1"),
					Pfds: map[string]models.Pfd{
						"pfd2": pfd2,
					},
				},
			},
		},
		{
			description: "Invalid App ID, should return ProblemDetails",
			afID:        "af1",
			transID:     "1",
			appID:       "app2",
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
				Body:   openapi.ProblemDetailsDataNotFound("Application ID not found"),
			},
		},
		{
			description: "Invalid PfdData, should return ProblemDetails",
			afID:        "af1",
			transID:     "1",
			appID:       "app1",
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
				Body:   openapi.ProblemDetailsDataNotFound(DetailNoPfdInfo),
			},
		},
	}

	Convey("Given AF, transaction and App ID, should partially update the PfdData", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)
				afPfdTans.AddExtAppID("app1")

				rsp := nefApp.Processor().PatchIndividualApplicationPFDManagement(tc.afID, tc.transID, tc.appID, tc.pfdData)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestValidatePfdManagement(t *testing.T) {
	testCases := []struct {
		description     string
		pfdManagement   *models.PfdManagement
		expectedProblem *models.ProblemDetails
		expectedReports map[string]models.PfdReport
	}{
		{
			description: "Valid",
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
			description: "Empty PfdDatas, should return ProblemDetails",
			pfdManagement: &models.PfdManagement{
				PfdDatas: map[string]models.PfdData{},
			},
			expectedProblem: openapi.ProblemDetailsDataNotFound(DetailNoPfdData),
			expectedReports: map[string]models.PfdReport{},
		},
		{
			description: "An appID is already provisioned, should mark in PfdReports",
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
			description: "None of the PFDs were created, should return ProblemDetails and mark in PfdReports",
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
			expectedProblem: openapi.ProblemDetailsSystemFailure("None of the PFDs were created"),
			expectedReports: map[string]models.PfdReport{
				string(models.FailureCode_APP_ID_DUPLICATED): {
					ExternalAppIds: []string{"app100"},
					FailureCode:    models.FailureCode_APP_ID_DUPLICATED,
				},
			},
		},
	}

	Convey("Given a PfdManagement along with its belonging AF and transaction ID, check its validity", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				afCtx := nefApp.Context().NewAfCtx("af1")
				nefApp.Context().AddAfCtx(afCtx)
				defer nefApp.Context().DeleteAfCtx("af1")
				afPfdTans := nefApp.Context().NewAfPfdTrans(afCtx)
				afCtx.AddPfdTrans(afPfdTans)
				afPfdTans.AddExtAppID("app100")

				rst := validatePfdManagement("af2", "1", tc.pfdManagement, nefApp.Context())
				So(rst, ShouldResemble, tc.expectedProblem)
				So(tc.pfdManagement.PfdReports, ShouldResemble, tc.expectedReports)
			})
		}
	})
}

func TestValidatePfdData(t *testing.T) {
	testCases := []struct {
		description    string
		pfdData        *models.PfdData
		expectedResult *models.ProblemDetails
	}{
		{
			description: "Valid",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
				},
			},
			expectedResult: nil,
		},
		{
			description: "Without ExternalAppId, should return ProblemDetails",
			pfdData: &models.PfdData{
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
				},
			},
			expectedResult: openapi.ProblemDetailsDataNotFound(DetailNoExtAppID),
		},
		{
			description: "Empty Pfds, should return ProblemDetails",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
			},
			expectedResult: openapi.ProblemDetailsDataNotFound(DetailNoPfd),
		},
		{
			description: "Without PfdID, should return ProblemDetails",
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
			expectedResult: openapi.ProblemDetailsDataNotFound(DetailNoPfdID),
		},
		{
			description: "FlowDescriptions, Urls and DomainNames are all empty, should return ProblemDetails",
			pfdData: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": {
						PfdId: "pfd1",
					},
				},
			},
			expectedResult: openapi.ProblemDetailsDataNotFound(DetailNoPfdInfo),
		},
	}

	Convey("Given a PfdData, check its validity", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				rst := validatePfdData(tc.pfdData, nefApp.Context(), false)
				So(rst, ShouldResemble, tc.expectedResult)
			})
		}
	})
}

func TestPatchModifyPfdData(t *testing.T) {
	testCases := []struct {
		description     string
		old             *models.PfdData
		new             *models.PfdData
		expectedProblem *models.ProblemDetails
		expectedResult  *models.PfdData
	}{
		{
			description: "Given a PfdData with non-existing appID, should append the Pfds to the PfdData",
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
			description: "Given a PfdData with existing appID, should update the PfdData",
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
			description: "Given a PfdData with existing appID and empty content, should delete the PfdData",
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
			description: "Given an invalid PfdData, should return ProblemDetails",
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
			expectedProblem: openapi.ProblemDetailsDataNotFound(DetailNoPfdInfo),
			expectedResult: &models.PfdData{
				ExternalAppId: "app1",
				Pfds: map[string]models.Pfd{
					"pfd1": pfd1,
				},
			},
		},
	}

	Convey("Given an old and new PfdData, should perform PATCH operation to update the old one", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				problemDetail := patchModifyPfdData(tc.old, tc.new)
				So(problemDetail, ShouldResemble, tc.expectedProblem)
				So(tc.old, ShouldResemble, tc.expectedResult)
			})
		}
	})
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
	gock.New("http://127.0.0.4:8000/nudr-dr/v1").
		Get("/application-data/pfds").
		// To Matching the request for both app1 and app2.
		// Should be clarified if there is a way to exact match multiple parameters with the same key.
		MatchParam("appId", "app1").
		Persist().
		Reply(http.StatusOK).
		JSON([]models.PfdDataForApp{pfdDataForApp1, pfdDataForApp2})

	gock.New("http://127.0.0.4:8000/nudr-dr/v1").
		Get("/application-data/pfds").
		MatchParam("appId", "app3").
		Persist().
		Reply(http.StatusNotFound).
		JSON(models.ProblemDetails{Status: http.StatusNotFound})
}

func initUDRDrGetPfdDataStub() {
	gock.New("http://127.0.0.4:8000/nudr-dr/v1").
		Get("/application-data/pfds/app1").
		Persist().
		Reply(http.StatusOK).
		JSON(pfdDataForApp1)

	gock.New("http://127.0.0.4:8000/nudr-dr/v1").
		Get("/application-data/pfds/app3").
		Persist().
		Reply(http.StatusNotFound).
		JSON(models.ProblemDetails{Status: http.StatusNotFound})
}

func initUDRDrDeletePfdDataStub() {
	gock.New("http://127.0.0.4:8000/nudr-dr/v1").
		Delete("/application-data/pfds/.*").
		Persist().
		Reply(http.StatusNoContent)
}

func initUDRDrPutPfdDataStub(statusCode int) {
	gock.New("http://127.0.0.4:8000/nudr-dr/v1").
		Put("/application-data/pfds/.*").
		Persist().
		Reply(statusCode).
		JSON(pfdDataForApp1)
}
