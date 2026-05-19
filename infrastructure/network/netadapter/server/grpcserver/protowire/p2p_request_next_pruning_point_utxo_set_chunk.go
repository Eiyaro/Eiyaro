package protowire

import (
	"github.com/Eiyaro/Eiyaro/app/appmessage"
	"github.com/pkg/errors"
)

func (x *EiyaroMessage_RequestNextPruningPointUtxoSetChunk) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "EiyaroMessage_RequestNextPruningPointUtxoSetChunk is nil")
	}
	return &appmessage.MsgRequestNextPruningPointUTXOSetChunk{}, nil
}

func (x *EiyaroMessage_RequestNextPruningPointUtxoSetChunk) fromAppMessage(_ *appmessage.MsgRequestNextPruningPointUTXOSetChunk) error {
	x.RequestNextPruningPointUtxoSetChunk = &RequestNextPruningPointUtxoSetChunkMessage{}
	return nil
}
