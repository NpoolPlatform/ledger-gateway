package ledger

import (
	"context"
	"fmt"
	"time"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	appusermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	"github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	ledgermgrpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"

	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

func CreateDeposit(
	ctx context.Context,
	appID, userID, langID, coinTypeID, amount, targetAppID, targetUserID string,
) (*ledger.Detail, error) {
	user, err := appusermwcli.GetUser(ctx, appID, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("target user not exist")
	}

	coin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: targetAppID,
		},
		CoinTypeID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: coinTypeID,
		},
	})
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid coin app_id:%v coin_type_id:%v", targetAppID, coinTypeID)
	}

	ioType := ledgermgrpb.IOType_Incoming
	ioSubtype := ledgermgrpb.IOSubType_Deposit
	ioExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","TargetAppID":"%v","TargetUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		appID,
		userID,
		targetAppID,
		targetUserID,
		coin.Name,
		amount,
		time.Now(),
	)
	createdAt := uint32(time.Now().Unix())

	err = ledgermwcli.BookKeeping(ctx, []*ledgermgrpb.DetailReq{
		{
			AppID:      &targetAppID,
			UserID:     &targetUserID,
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
		CoinTypeID:   coinTypeID,
		CoinName:     coin.Name,
		DisplayNames: coin.DisplayNames,
		CoinLogo:     coin.Logo,
		CoinUnit:     coin.Unit,
		IOType:       ioType,
		IOSubType:    ioSubtype,
		Amount:       amount,
		IOExtra:      ioExtra,
		CreatedAt:    createdAt,
	}, nil
}
