package deposit

import (
	"context"
	"fmt"
	"time"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	ledgerpb "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"

	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	statementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/statement"
)

func (h *Handler) CreateDeposit(ctx context.Context) (*npool.Statement, error) {
	coin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.TargetAppID,
		},
		CoinTypeID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.CoinTypeID,
		},
	})
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid coin app_id:%v coin_type_id:%v", *h.TargetAppID, *h.CoinTypeID)
	}

	ioExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","TargetAppID":"%v","TargetUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		*h.AppID,
		*h.UserID,
		*h.TargetAppID,
		*h.TargetUserID,
		coin.Name,
		*h.Amount,
		time.Now(),
	)

	ioType := ledgerpb.IOType_Incoming
	ioSubtype := ledgerpb.IOSubType_Deposit
	infos, err := ledgermwcli.CreateStatements(ctx, []*statementpb.StatementReq{
		{
			AppID:      h.AppID,
			UserID:     h.TargetUserID,
			CoinTypeID: h.CoinTypeID,
			IOType:     &ioType,
			IOSubType:  &ioSubtype,
			Amount:     h.Amount,
			IOExtra:    &ioExtra,
		},
	})
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, nil
	}
	info := infos[0]

	return &npool.Statement{
		CoinTypeID:   *h.CoinTypeID,
		CoinName:     coin.Name,
		DisplayNames: coin.DisplayNames,
		CoinLogo:     coin.Logo,
		CoinUnit:     coin.Unit,
		IOType:       info.IOType,
		IOSubType:    info.IOSubType,
		Amount:       info.Amount,
		IOExtra:      info.IOExtra,
		CreatedAt:    info.CreatedAt,
	}, nil
}
