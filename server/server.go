package server

import (
	"errors"
	"gitlab.com/lightmeter/controlcenter/api"
	httpauth "gitlab.com/lightmeter/controlcenter/httpauth"
	auth "gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/httpsettings"
	"gitlab.com/lightmeter/controlcenter/i18n"
	"gitlab.com/lightmeter/controlcenter/newsletter"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/staticdata"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/workspace"
	"log"
	"net"
	"net/http"
	"time"
)

type HttpServer struct {
	Workspace          *workspace.Workspace
	WorkspaceDirectory string
	Timezone           *time.Location
	Address            string
	FrontendDev        bool
}

func (s *HttpServer) Start() error {
	if s.Workspace == nil {
		return errorutil.Wrap(errors.New("Workspace is nil"))
	}

	if s.WorkspaceDirectory == "" {
		return errorutil.Wrap(errors.New("WorkspaceDirectory is empty string"))
	}

	if s.Timezone == nil {
		return errorutil.Wrap(errors.New("Timezone is nil"))
	}

	if s.Address == "" {
		return errorutil.Wrap(errors.New("Address is empty string"))
	}

	initialSetupSettings := settings.NewInitialSetupSettings(newsletter.NewSubscriber("https://phplist.lightmeter.io/"))

	writer, reader := s.Workspace.SettingsAcessors()

	setup := httpsettings.NewSettings(writer, reader, initialSetupSettings, s.Workspace.NotificationCenter)

	auth := auth.NewAuthenticator(s.Workspace.Auth(), s.WorkspaceDirectory)

	mux := http.NewServeMux()

	i18nService := i18n.NewService(po.DefaultCatalog)

	chain := httpmiddleware.WithDefaultStackWithoutAuth()
	mux.Handle("/language/metadata", chain.WithError(httpmiddleware.CustomHTTPHandler(i18nService.LanguageMetaDataHandler)))

	mux.Handle("/", http.StripPrefix("/", http.FileServer(staticdata.HttpAssets)))

	exposeApiExplorer(mux)

	exposeProfiler(mux)

	dashboard, err := s.Workspace.Dashboard()

	if err != nil {
		return errorutil.Wrap(err, "Error creating dashboard")
	}

	insightsFetcher := s.Workspace.InsightsFetcher()

	api.HttpDashboard(auth, mux, s.Timezone, dashboard)
	api.HttpInsights(auth, mux, s.Timezone, insightsFetcher)

	setup.HttpSetup(mux, auth)

	httpauth.HttpAuthenticator(mux, auth)

	server := http.Server{Handler: wrap(mux)}

	ln, err := net.Listen("tcp", s.Address)

	if err != nil {
		return errorutil.Wrap(err)
	}

	log.Printf("Lightmeter ControlCenter is running on http://%s", ln.Addr().String())

	return server.Serve(ln)
}
