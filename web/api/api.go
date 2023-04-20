package api

import (
	"github.com/caofujiang/winchaos/transport"
	chaosweb "github.com/caofujiang/winchaos/web"
	"github.com/caofujiang/winchaos/web/handler"
	"github.com/caofujiang/winchaos/web/server"
)

type API struct {
	chaosweb.APiServer
	//ready func(http.HandlerFunc) http.HandlerFunc
}

// community just use http
func NewAPI() *API {
	return &API{
		server.NewHttpServer(),
	}
}

func (api *API) Register(transportClient *transport.TransportClient) error {
	chaosbladeHandler := NewServerRequestHandler(handler.NewChaosbladeHandler(transportClient))
	if err := api.RegisterHandler("chaosblade", chaosbladeHandler); err != nil {
		return err
	}

	pingHandler := NewServerRequestHandler(handler.NewPingHandler())
	if err := api.RegisterHandler("ping", pingHandler); err != nil {
		return err
	}

	uninstallHandler := NewServerRequestHandler(handler.NewUninstallInstallHandler(transportClient))
	if err := api.RegisterHandler("uninstall", uninstallHandler); err != nil {
		return err
	}

	updateApplicationHandler := NewServerRequestHandler(handler.NewUpdateApplicationHandler())
	if err := api.RegisterHandler("updateApplication", updateApplicationHandler); err != nil {
		return err
	}

	return nil
}
