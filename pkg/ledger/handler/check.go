package handler

import "github.com/NpoolPlatform/go-service-framework/pkg/wlog"

func (h *Handler) CheckStartEndAt() error {
	if (h.StartAt != nil && h.EndAt != nil) && *h.StartAt > *h.EndAt {
		return wlog.Errorf("invalid startat and endat")
	}
	return nil
}
