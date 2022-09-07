package ledger

import (
	"context"
	"fmt"
	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/v2"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	"github.com/NpoolPlatform/message/npool"

	appusermgrcli "github.com/NpoolPlatform/appuser-manager/pkg/client/appuser"
	appusermgrpb "github.com/NpoolPlatform/message/npool/appuser/mgr/v2/appuser"

	"github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	ledgermgrpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"
	"time"
)

func CreateDeposit(ctx context.Context, userID, appID, coinTypeID, amount, depositAppID, depositUserID string) (*ledger.Detail, error) {
	depositUser, err := appusermgrcli.GetAppUserOnly(ctx, &appusermgrpb.Conds{
		AppID: &npool.StringVal{
			Op:    cruder.EQ,
			Value: depositAppID,
		},
		ID: &npool.StringVal{
			Op:    cruder.EQ,
			Value: depositUserID,
		},
	})
	if err != nil {
		return nil, err
	}
	if depositUser == nil {
		return nil, fmt.Errorf("deposit user not found")
	}

	coin, err := coininfocli.GetCoinInfo(ctx, coinTypeID)
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid coin")
	}

	ioType := ledgermgrpb.IOType_Incoming
	ioSubtype := ledgermgrpb.IOSubType_Deposit
	ioExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","DepositAppID":"%v","DepositUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		appID,
		userID,
		depositAppID,
		depositUserID,
		coin.Name,
		amount,
		time.Now(),
	)
	createdAt := uint32(time.Now().Unix())

	err = ledgermwcli.BookKeeping(ctx, []*ledgermgrpb.DetailReq{
		{
			AppID:      &depositAppID,
			UserID:     &depositUserID,
			CoinTypeID: &coinTypeID,
			IOType:     &ioType,
			IOSubType:  &ioSubtype,
			Amount:     &amount,
			IOExtra:    &ioExtra,
			CreatedAt:  &createdAt,
		},
	})
	if err != nil {
		return nil, err
	}

	return &ledger.Detail{
		CoinTypeID: coinTypeID,
		CoinName:   coin.Name,
		CoinLogo:   coin.Logo,
		CoinUnit:   coin.Unit,
		IOType:     ioType,
		IOSubType:  ioSubtype,
		Amount:     amount,
		IOExtra:    ioExtra,
		CreatedAt:  createdAt,
	}, nil
}
