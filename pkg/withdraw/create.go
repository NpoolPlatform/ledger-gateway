package withdraw

import (
	"context"
	"fmt"

	pltfaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/platform"
	useraccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/user"
	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	usercodemwcli "github.com/NpoolPlatform/basal-middleware/pkg/client/usercode"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	currencymwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"
	withdrawmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/withdraw"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	pltfaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/platform"
	useraccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/user"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	usercodemwpb "github.com/NpoolPlatform/message/npool/basal/mw/v1/usercode"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	coinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin"
	currencymwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	withdrawmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw"
	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type createHandler struct {
	*Handler
	user            *usermwpb.User
	withdrawAccount *useraccmwpb.Account
	withdrawAmount  decimal.Decimal
	coin            *coinmwpb.Coin
	appCoin         *appcoinmwpb.Coin
}

func (h *createHandler) checkUser(ctx context.Context) error {
	user, err := usermwcli.GetUser(ctx, *h.AppID, *h.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalid user")
	}
	if user.State != basetypes.KycState_Approved {
		return fmt.Errorf("kyc not approved, user id(%v)", h.UserID)
	}
	if *h.AccountType == basetypes.SignMethod_Google {
		h.Account = &user.GoogleSecret
	}
	h.user = user
	return nil
}

func (h *createHandler) verifyUserCode(ctx context.Context) error {
	return usercodemwcli.VerifyUserCode(ctx, &usercodemwpb.VerifyUserCodeRequest{
		Prefix:      basetypes.Prefix_PrefixUserCode.String(),
		AppID:       *h.AppID,
		Account:     *h.Account,
		AccountType: *h.AccountType,
		UsedFor:     basetypes.UsedFor_Withdraw,
		Code:        *h.VerificationCode,
	})
}

func (h *createHandler) checkWithdrawAmount(ctx context.Context) error {
	ledger, err := ledgermwcli.GetLedgerOnly(ctx, &ledgermwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
	})
	if err != nil {
		return err
	}
	if ledger == nil {
		return fmt.Errorf("ledger not exist")
	}
	spendable, err := decimal.NewFromString(ledger.Spendable)
	if err != nil {
		return err
	}
	if spendable.Cmp(h.withdrawAmount) < 0 {
		return fmt.Errorf("insufficient funds")
	}
	maxAmount, err := decimal.NewFromString(h.appCoin.MaxAmountPerWithdraw)
	if err != nil {
		return err
	}
	if h.withdrawAmount.Cmp(maxAmount) > 0 {
		return fmt.Errorf("overflow")
	}
	return nil
}

func (h *createHandler) getUserAccount(ctx context.Context) error {
	account, err := useraccmwcli.GetAccountOnly(ctx, &useraccmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
		AccountID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AccountID},
		Active:     &basetypes.BoolVal{Op: cruder.EQ, Value: true},
		Blocked:    &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		UsedFor:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(basetypes.AccountUsedFor_UserWithdraw)},
	})
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("invalid account")
	}

	h.withdrawAccount = account
	if !h.appCoin.CheckNewAddressBalance {
		return nil
	}

	bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    h.coin.Name,
		Address: account.Address,
	})
	if err != nil {
		return err
	}
	if bal == nil {
		return fmt.Errorf("can not get balance")
	}

	return nil
}

func (h *createHandler) getPlatformAccount(ctx context.Context) error {
	account, err := pltfaccmwcli.GetAccountOnly(ctx, &pltfaccmwpb.Conds{
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
		UsedFor:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(basetypes.AccountUsedFor_UserBenefitHot)},
		Active:     &basetypes.BoolVal{Op: cruder.EQ, Value: true},
		Backup:     &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		Blocked:    &basetypes.BoolVal{Op: cruder.EQ, Value: false},
	})
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("invalid hot wallet account")
	}

	bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    h.coin.Name,
		Address: account.Address,
	})
	if err != nil {
		return err
	}
	if bal == nil {
		return fmt.Errorf("can not get balance")
	}
	return nil
}

func (h *createHandler) checkCoin(ctx context.Context) error {
	coin, err := coininfocli.GetCoin(ctx, *h.CoinTypeID)
	if coin == nil {
		return fmt.Errorf("coin not found %v", *h.CoinTypeID)
	}
	if err != nil {
		return err
	}
	if coin.Disabled {
		return fmt.Errorf("coin disabled")
	}
	appCoin, err := appcoinmwcli.GetCoin(ctx, *h.CoinTypeID)
	if err != nil {
		return err
	}
	if appCoin == nil {
		return fmt.Errorf("app coin not found %v", *h.CoinTypeID)
	}
	if appCoin.Disabled {
		return fmt.Errorf("app coin disabled")
	}
	return nil
}

func (h *createHandler) checkWithdrawFeeAmount(ctx context.Context) error {
	feeAmount, err := decimal.NewFromString(h.appCoin.WithdrawFeeAmount)
	if err != nil {
		return err
	}
	if feeAmount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid fee amount")
	}

	if !h.appCoin.WithdrawFeeByStableUSD {
		if h.withdrawAmount.Cmp(feeAmount) <= 0 {
			return fmt.Errorf("invalid amount")
		}
	}

	curr, err := currencymwcli.GetCurrencyOnly(ctx, &currencymwpb.Conds{
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: h.coin.ID},
	})
	if err != nil {
		return err
	}
	value, err := decimal.NewFromString(curr.MarketValueLow)
	if err != nil {
		return err
	}
	if value.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid coin price")
	}
	feeAmount = feeAmount.Div(value)
	if h.withdrawAmount.Cmp(feeAmount) <= 0 {
		return fmt.Errorf("invalid amount")
	}
	return nil
}

func (h *Handler) CreateWithdraw(ctx context.Context) (*npool.Withdraw, error) {
	handler := &createHandler{
		Handler: h,
	}
	_amount, err := decimal.NewFromString(*h.Amount)
	if err != nil {
		return nil, err
	}
	handler.withdrawAmount = _amount
	if err := handler.checkUser(ctx); err != nil {
		return nil, err
	}
	if err := handler.verifyUserCode(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkCoin(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkWithdrawAmount(ctx); err != nil {
		return nil, err
	}
	if err := handler.getUserAccount(ctx); err != nil {
		return nil, err
	}
	if err := handler.getPlatformAccount(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkWithdrawFeeAmount(ctx); err != nil {
		return nil, err
	}

	id := uuid.NewString()
	if h.ID == nil {
		h.ID = &id
	}

	if _, err := withdrawmwcli.CreateWithdraw(ctx, &withdrawmwpb.WithdrawReq{
		ID:         h.ID,
		AppID:      h.AppID,
		UserID:     h.UserID,
		CoinTypeID: h.CoinTypeID,
		AccountID:  h.AccountID,
		Address:    &handler.withdrawAccount.Address,
		Amount:     h.Amount,
	}); err != nil {
		return nil, err
	}

	info, err := h.GetWithdraw(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("invalid withdraw")
	}
	return info, nil
}
