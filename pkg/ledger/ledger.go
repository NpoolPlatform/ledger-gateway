package ledger

import (
	"context"
	"fmt"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
)

func GetGenerals(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.General, uint32, error) {
	return nil, 0, fmt.Errorf("NOT IMPLEMENTED")
}

func GetInterval(ctx context.Context, appID, userID string, start, end uint32, offset, limit int32) ([]*npool.General, uint32, error) {
	return nil, 0, fmt.Errorf("NOT IMPLEMENTED")
}

func GetDetails(ctx context.Context, appID, userID string, start, end uint32, offset, limit int32) ([]*npool.Detail, uint32, error) {
	return nil, 0, fmt.Errorf("NOT IMPLEMENTED")
}
