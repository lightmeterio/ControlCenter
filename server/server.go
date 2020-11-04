package server

import (
	"errors"
	"gitlab.com/lightmeter/controlcenter/api"
	"gitlab.com/lightmeter/controlcenter/httpauth"
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

	mux := http.NewServeMux()

	mux.Handle("/", i18n.DefaultWrap(http.FileServer(staticdata.HttpAssets), staticdata.HttpAssets, po.DefaultCatalog))

	exposeApiExplorer(mux)

	exposeProfiler(mux)

	dashboard, err := s.Workspace.Dashboard()

	if err != nil {
		return errorutil.Wrap(err, "Error creating dashboard")
	}

	insightsFetcher := s.Workspace.InsightsFetcher()

	api.HttpDashboard(mux, s.Timezone, dashboard)
	api.HttpInsights(mux, s.Timezone, insightsFetcher)

	setup.SetupMux(mux)

	// Some paths that don't require authentication
	// That's what people nowadays call a "allow list".
	publicPaths := []string{
		"/img",
		"/css",
		"/fonts",
		"/js",
		"/3rd",
		"/debug",
	}

	setup.HttpSettingsPage(mux)

	authWrapper := httpauth.Serve(mux, s.Workspace.Auth(), s.WorkspaceDirectory, publicPaths)

	server := http.Server{Handler: authWrapper}

	ln, err := net.Listen("tcp", s.Address)

	if err != nil {
		return errorutil.Wrap(err)
	}

	log.Printf("Lightmeter ControlCenter is running on http://%s", ln.Addr().String())

	return server.Serve(ln)
}
