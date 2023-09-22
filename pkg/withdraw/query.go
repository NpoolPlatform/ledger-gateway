package withdraw

import (
	"context"

	useraccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	servicename "github.com/NpoolPlatform/ledger-gateway/pkg/servicename"
	withdrawmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/withdraw"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	useraccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/user"
	reviewtypes "github.com/NpoolPlatform/message/npool/basetypes/review/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw"
	withdrawmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw"
	reviewmwcli "github.com/NpoolPlatform/review-middleware/pkg/client/review"
)

type queryHandler struct {
	*Handler
	withdraws      []*withdrawmwpb.Withdraw
	accounts       map[string]*useraccmwpb.Account
	appCoins       map[string]*appcoinmwpb.Coin
	reviewMessages map[string]string
	infos          []*npool.Withdraw
}

func (h *queryHandler) getAccounts(ctx context.Context) error {
	accountIDs := []string{}
	for _, withdraw := range h.withdraws {
		accountIDs = append(accountIDs, withdraw.AccountID)
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
		return err
	}
	for _, account := range accounts {
		h.accounts[account.AccountID] = account
	}
	return nil
}

func (h *queryHandler) getCoins(ctx context.Context) error {
	coinTypeIDs := []string{}
	for _, withdraw := range h.withdraws {
		coinTypeIDs = append(coinTypeIDs, withdraw.CoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}
	for _, coin := range coins {
		h.appCoins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *queryHandler) getReviews(ctx context.Context) error {
	withdrawIDs := []string{}
	for _, withdraw := range h.withdraws {
		withdrawIDs = append(withdrawIDs, withdraw.ID)
	}
	reviews, err := reviewmwcli.GetObjectReviews(
		ctx,
		*h.AppID,
		servicename.ServiceDomain,
		withdrawIDs,
		reviewtypes.ReviewObjectType_ObjectWithdrawal,
	)
	if err != nil {
		return err
	}
	for _, r := range reviews {
		if r.State == reviewtypes.ReviewState_Rejected {
			h.reviewMessages[r.ObjectID] = r.Message
		}
	}
	return nil
}

func (h *queryHandler) formalize() {
	for _, withdraw := range h.withdraws {
		coin, ok := h.appCoins[withdraw.CoinTypeID]
		if !ok {
			continue
		}

		address := withdraw.Address
		labels := []string{}

		account, ok := h.accounts[withdraw.AccountID]
		if ok {
			labels = account.Labels
			address = account.Address
		}

		h.infos = append(h.infos, &npool.Withdraw{
			ID:            withdraw.ID,
			AppID:         withdraw.AppID,
			UserID:        withdraw.UserID,
			CoinTypeID:    withdraw.CoinTypeID,
			CoinName:      coin.CoinName,
			DisplayNames:  coin.DisplayNames,
			CoinLogo:      coin.Logo,
			CoinUnit:      coin.Unit,
			Amount:        withdraw.Amount,
			CreatedAt:     withdraw.CreatedAt,
			Address:       address,
			AddressLabels: labels,
			State:         withdraw.State,
			Message:       h.reviewMessages[withdraw.ID],
		})
	}
}

func (h *Handler) GetWithdraws(ctx context.Context) ([]*npool.Withdraw, uint32, error) {
	conds := &withdrawmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	withdraws, total, err := withdrawmwcli.GetWithdraws(ctx, conds, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(withdraws) == 0 {
		return nil, total, nil
	}

	handler := &queryHandler{
		Handler:        h,
		withdraws:      withdraws,
		accounts:       map[string]*useraccmwpb.Account{},
		appCoins:       map[string]*appcoinmwpb.Coin{},
		reviewMessages: map[string]string{},
	}

	if err := handler.getAccounts(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getReviews(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()

	return handler.infos, total, nil
}

func (h *Handler) GetWithdraw(ctx context.Context) (*npool.Withdraw, error) {
	withdraw, err := withdrawmwcli.GetWithdraw(ctx, *h.ID)
	if err != nil {
		return nil, err
	}
	if withdraw == nil {
		return nil, nil
	}

	handler := &queryHandler{
		Handler:        h,
		withdraws:      []*withdrawmwpb.Withdraw{withdraw},
		accounts:       map[string]*useraccmwpb.Account{},
		appCoins:       map[string]*appcoinmwpb.Coin{},
		reviewMessages: map[string]string{},
	}

	if err := handler.getAccounts(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCoins(ctx); err != nil {
		return nil, err
	}
	if err := handler.getReviews(ctx); err != nil {
		return nil, err
	}

	handler.formalize()
	if len(handler.infos) == 0 {
		return nil, nil
	}

	return handler.infos[0], nil
}
