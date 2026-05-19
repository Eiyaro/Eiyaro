package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Eiyaro/Eiyaro/infrastructure/network/netadapter/server/grpcserver/protowire"
)

var commandTypes = []reflect.Type{
	reflect.TypeFor[protowire.EiyaroMessage_AddPeerRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetConnectedPeerInfoRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetPeerAddressesRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetCurrentNetworkRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetInfoRequest](),

	reflect.TypeFor[protowire.EiyaroMessage_GetBlockRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetBlockByTransactionIdRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetBlocksRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetHeadersRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetBlockCountRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetBlockDagInfoRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetSelectedTipHashRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetVirtualSelectedParentBlueScoreRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetVirtualSelectedParentChainFromBlockRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_ResolveFinalityConflictRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_EstimateNetworkHashesPerSecondRequest](),

	reflect.TypeFor[protowire.EiyaroMessage_GetBlockTemplateRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_SubmitBlockRequest](),

	reflect.TypeFor[protowire.EiyaroMessage_GetMempoolEntryRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetMempoolEntriesRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetMempoolEntriesByAddressesRequest](),

	reflect.TypeFor[protowire.EiyaroMessage_SubmitTransactionRequest](),

	reflect.TypeFor[protowire.EiyaroMessage_GetUtxosByAddressesRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetBalanceByAddressRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_GetCoinSupplyRequest](),

	reflect.TypeFor[protowire.EiyaroMessage_BanRequest](),
	reflect.TypeFor[protowire.EiyaroMessage_UnbanRequest](),
}

type commandDescription struct {
	name       string
	parameters []*parameterDescription
	typeof     reflect.Type
}

type parameterDescription struct {
	name   string
	typeof reflect.Type
}

func commandDescriptions() []*commandDescription {
	commandDescriptions := make([]*commandDescription, len(commandTypes))

	for i, commandTypeWrapped := range commandTypes {
		commandType := unwrapCommandType(commandTypeWrapped)

		name := strings.TrimSuffix(commandType.Name(), "RequestMessage")
		numFields := commandType.NumField()

		var parameters []*parameterDescription
		for i := range numFields {
			field := commandType.Field(i)

			if !isFieldExported(field) {
				continue
			}

			parameters = append(parameters, &parameterDescription{
				name:   field.Name,
				typeof: field.Type,
			})
		}
		commandDescriptions[i] = &commandDescription{
			name:       name,
			parameters: parameters,
			typeof:     commandTypeWrapped,
		}
	}

	return commandDescriptions
}

func (cd *commandDescription) help() string {
	sb := &strings.Builder{}
	sb.WriteString(cd.name)
	for _, parameter := range cd.parameters {
		_, _ = fmt.Fprintf(sb, " [%s]", parameter.name)
	}
	return sb.String()
}
