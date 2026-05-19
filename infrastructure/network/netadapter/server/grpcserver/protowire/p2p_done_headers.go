package protowire

import (
	"github.com/Eiyaro/Eiyaro/app/appmessage"
	"github.com/pkg/errors"
)

func (x *EiyaroMessage_DoneHeaders) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "EiyaroMessage_DoneHeaders is nil")
	}
	return &appmessage.MsgDoneHeaders{}, nil
}

func (x *EiyaroMessage_DoneHeaders) fromAppMessage(_ *appmessage.MsgDoneHeaders) error {
	return nil
}
