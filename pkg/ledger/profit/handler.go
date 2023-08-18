package profit

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/ledger-gateway/pkg/ledger/handler"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
)

type Handler struct {
	*handler.Handler
	IOType    *basetypes.IOType
	IOSubType *basetypes.IOSubType
}

func NewHandler(ctx context.Context, options ...interface{}) (*Handler, error) {
	_handler, err := handler.NewHandler(ctx, options...)
	if err != nil {
		return nil, err
	}
	h := &Handler{
		Handler: _handler,
	}

	for _, opt := range options {
		_opt, ok := opt.(func(context.Context, *Handler) error)
		if !ok {
			continue
		}
		if err := _opt(ctx, h); err != nil {
			return nil, err
		}
	}
	return h, nil
}

func WithIOtype(_type *basetypes.IOType, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if _type == nil {
			if must {
				return fmt.Errorf("invalid io type")
			}
			return nil
		}
		flag := false
		for ioType := range basetypes.IOType_value {
			if ioType == _type.String() && ioType != basetypes.IOType_DefaultType.String() {
				flag = true
			}
		}
		if !flag {
			return fmt.Errorf("invalid io type %v", *_type)
		}
		h.IOType = _type
		return nil
	}
}

func WithIOSubType(_type *basetypes.IOSubType, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if _type == nil {
			if must {
				return fmt.Errorf("invalid io sub type")
			}
			return nil
		}

		flag := false
		for ioSubType := range basetypes.IOSubType_value {
			if ioSubType == _type.String() && ioSubType != basetypes.IOSubType_DefaultSubType.String() {
				flag = true
			}
		}
		if !flag {
			return fmt.Errorf("invalid io sub type %v", *_type)
		}
		h.IOSubType = _type
		return nil
	}
}
