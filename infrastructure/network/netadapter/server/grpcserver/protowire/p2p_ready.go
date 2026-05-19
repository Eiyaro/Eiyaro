package protowire

import (
	"github.com/Eiyaro/Eiyaro/app/appmessage"
	"github.com/pkg/errors"
)

func (x *EiyaroMessage_Ready) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "EiyaroMessage_Ready is nil")
	}
	return &appmessage.MsgReady{}, nil
}

func (x *EiyaroMessage_Ready) fromAppMessage(_ *appmessage.MsgReady) error {
	return nil
}
