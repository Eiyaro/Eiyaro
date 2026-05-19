package rpchandlers

import (
	"github.com/Eiyaro/Eiyaro/app/appmessage"
	"github.com/Eiyaro/Eiyaro/app/rpc/rpccontext"
	"github.com/Eiyaro/Eiyaro/infrastructure/network/netadapter/router"
)

// HandleGetCurrentNetwork handles the respectively named RPC command
func HandleGetCurrentNetwork(context *rpccontext.Context, _ *router.Router, _ appmessage.Message) (appmessage.Message, error) {
	response := appmessage.NewGetCurrentNetworkResponseMessage(context.Config.ActiveNetParams.Net.String())
	return response, nil
}
