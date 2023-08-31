package withdraw

import (
	"context"

	withdrawmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/withdraw"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw"
	withdrawpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw"

	useraccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/user"
	useraccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/user"

	servicename "github.com/NpoolPlatform/ledger-gateway/pkg/servicename"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	reviewpb "github.com/NpoolPlatform/message/npool/review/mw/v2/review"
	reviewcli "github.com/NpoolPlatform/review-middleware/pkg/client/review"
)

//nolint
func (h *Handler) GetWithdraws(ctx context.Context) ([]*npool.Withdraw, uint32, error) {
	withdraws, total, err := withdrawmwcli.GetWithdraws(ctx, &withdrawpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
	}, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(withdraws) == 0 {
		return []*npool.Withdraw{}, 0, nil
	}

	accountIDs := []string{}
	coinTypeIDs := []string{}
	withdrawIDs := []string{}
	for _, info := range withdraws {
		accountIDs = append(accountIDs, info.AccountID)
		coinTypeIDs = append(coinTypeIDs, info.CoinTypeID)
		withdrawIDs = append(withdrawIDs, info.ID)
	}

	conds := &useraccmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UsedFor:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(basetypes.AccountUsedFor_UserWithdraw)},
		AccountIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: accountIDs},
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}

	accounts, _, err := useraccmwcli.GetAccounts(ctx, conds, 0, int32(len(accountIDs)))
	if err != nil {
		return nil, 0, err
	}
	if err != nil {
		return nil, 0, err
	}

	accMap := map[string]*useraccmwpb.Account{}
	for _, acc := range accounts {
		accMap[acc.AccountID] = acc
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	reviews, err := reviewcli.GetObjectReviews(
		ctx,
		*h.AppID,
		servicename.ServiceDomain,
		withdrawIDs,
		reviewpb.ReviewObjectType_ObjectWithdrawal,
	)
	if err != nil {
		return nil, 0, err
	}
	messageMap := map[string]string{}
	for _, r := range reviews {
		if r.State == reviewpb.ReviewState_Rejected {
			messageMap[r.ObjectID] = r.Message
		}
	}

	infos := []*npool.Withdraw{}
	for _, info := range withdraws {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			continue
		}

		address := info.Address
		labels := []string{}

		wacc, ok := accMap[info.AccountID]
		if ok {
			labels = wacc.Labels
			address = wacc.Address
		}

		infos = append(infos, &npool.Withdraw{
			CoinTypeID:    info.CoinTypeID,
			CoinName:      coin.CoinName,
			DisplayNames:  coin.DisplayNames,
			CoinLogo:      coin.Logo,
			CoinUnit:      coin.Unit,
			Amount:        info.Amount,
			CreatedAt:     info.CreatedAt,
			Address:       address,
			AddressLabels: labels,
			State:         info.State,
			Message:       messageMap[info.ID],
		})
	}
	return infos, total, nil
}
