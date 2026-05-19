package protowire

import (
	"github.com/Eiyaro/Eiyaro/app/appmessage"
	"github.com/pkg/errors"
)

func (x *EiyaroMessage_ShutDownRequest) toAppMessage() (appmessage.Message, error) {
	return &appmessage.ShutDownRequestMessage{}, nil
}

func (x *EiyaroMessage_ShutDownRequest) fromAppMessage(_ *appmessage.ShutDownRequestMessage) error {
	x.ShutDownRequest = &ShutDownRequestMessage{}
	return nil
}

func (x *EiyaroMessage_ShutDownResponse) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "EiyaroMessage_ShutDownResponse is nil")
	}
	return x.ShutDownResponse.toAppMessage()
}

func (x *EiyaroMessage_ShutDownResponse) fromAppMessage(message *appmessage.ShutDownResponseMessage) error {
	var err *RPCError
	if message.Error != nil {
		err = &RPCError{Message: message.Error.Message}
	}
	x.ShutDownResponse = &ShutDownResponseMessage{
		Error: err,
	}
	return nil
}

func (x *ShutDownResponseMessage) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "ShutDownResponseMessage is nil")
	}
	rpcErr, err := x.Error.toAppMessage()
	// Error is an optional field
	if err != nil && !errors.Is(err, errorNil) {
		return nil, err
	}
	return &appmessage.ShutDownResponseMessage{
		Error: rpcErr,
	}, nil
}
