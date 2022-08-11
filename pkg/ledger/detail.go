package ledger

import (
	"context"
	"fmt"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
)

func GetDetails(ctx context.Context, appID, userID string, start, end uint32, offset, limit int32) ([]*npool.Detail, uint32, error) {
	details, total, err := ledgermwcli.GetIntervalDetails(ctx, appID, userID, start, end, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	coins, err := coininfocli.GetCoinInfos(ctx, cruder.NewFilterConds())
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*coininfopb.CoinInfo{}
	for _, coin := range coins {
		coinMap[coin.ID] = coin
	}

	infos := []*npool.Detail{}
	for _, info := range details {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		infos = append(infos, &npool.Detail{
			CoinTypeID: info.CoinTypeID,
			CoinName:   coin.Name,
			CoinLogo:   coin.Logo,
			CoinUnit:   coin.Unit,
			IOType:     info.IOType,
			IOSubType:  info.IOSubType,
			Amount:     info.Amount,
			IOExtra:    info.IOExtra,
			CreatedAt:  info.CreatedAt,
		})
	}

	return infos, total, nil
}
