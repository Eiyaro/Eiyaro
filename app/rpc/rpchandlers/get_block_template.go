package rpchandlers

import (
	"github.com/Eiyaro/Eiyaro/app/appmessage"
	"github.com/Eiyaro/Eiyaro/app/rpc/rpccontext"
	"github.com/Eiyaro/Eiyaro/domain/consensus/model/externalapi"
	"github.com/Eiyaro/Eiyaro/domain/consensus/utils/txscript"
	"github.com/Eiyaro/Eiyaro/infrastructure/network/netadapter/router"
	"github.com/Eiyaro/Eiyaro/util"
	"github.com/Eiyaro/Eiyaro/version"
)

// HandleGetBlockTemplate handles the respectively named RPC command
func HandleGetBlockTemplate(context *rpccontext.Context, _ *router.Router, request appmessage.Message) (appmessage.Message, error) {
	getBlockTemplateRequest := request.(*appmessage.GetBlockTemplateRequestMessage)

	payAddress, err := util.DecodeAddress(getBlockTemplateRequest.PayAddress, context.Config.ActiveNetParams.Prefix)
	if err != nil {
		errorMessage := &appmessage.GetBlockTemplateResponseMessage{}
		errorMessage.Error = appmessage.RPCErrorf("Could not decode address: %s", err)
		return errorMessage, nil
	}

	scriptPublicKey, err := txscript.PayToAddrScript(payAddress)
	if err != nil {
		return nil, err
	}

	coinbaseData := &externalapi.DomainCoinbaseData{ScriptPublicKey: scriptPublicKey, ExtraData: []byte(version.Version() + "/" + getBlockTemplateRequest.ExtraData)}

	templateBlock, isNearlySynced, err := context.Domain.MiningManager().GetBlockTemplate(coinbaseData)
	if err != nil {
		return nil, err
	}

	rpcBlock := appmessage.DomainBlockToRPCBlock(templateBlock)

	// Eiyaro: Allow mining on isolated nodes (no peers) if nearly synced
	// This enables the first node to mine without requiring network peers
	hasPeers := context.ProtocolManager.Context().HasPeers()
	isSynced := isNearlySynced && (hasPeers || true) // Always allow mining if nearly synced, regardless of peers
	log.Debugf("GetBlockTemplate: hasPeers=%v, isNearlySynced=%v, isSynced=%v", hasPeers, isNearlySynced, isSynced)
	return appmessage.NewGetBlockTemplateResponseMessage(rpcBlock, isSynced), nil
}
