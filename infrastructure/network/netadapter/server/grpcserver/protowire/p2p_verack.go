package protowire

import (
	"github.com/Eiyaro/Eiyaro/app/appmessage"
	"github.com/pkg/errors"
)

func (x *EiyaroMessage_Verack) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "EiyaroMessage_Verack is nil")
	}
	return &appmessage.MsgVerAck{}, nil
}

func (x *EiyaroMessage_Verack) fromAppMessage(_ *appmessage.MsgVerAck) error {
	return nil
}
