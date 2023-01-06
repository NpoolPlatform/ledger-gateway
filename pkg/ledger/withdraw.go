package ledger

import (
	"context"
	"fmt"
	"time"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	"github.com/NpoolPlatform/message/npool/third/mgr/v1/usedfor"

	"github.com/shopspring/decimal"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermgrwithdrawcli "github.com/NpoolPlatform/ledger-manager/pkg/client/withdraw"
	ledgermgrdetailpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"
	ledgermgrwithdrawpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/withdraw"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	ledgermgrgeneralcli "github.com/NpoolPlatform/ledger-manager/pkg/client/general"
	ledgermgrgeneralpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/general"

	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"

	txmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/tx"
	txmgrpb "github.com/NpoolPlatform/message/npool/chain/mgr/v1/tx"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/appcoin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/appcoin"

	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	useraccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/user"
	accountmgrpb "github.com/NpoolPlatform/message/npool/account/mgr/v1/account"
	useraccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/user"

	pltfaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/platform"
	pltfaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/platform"

	reviewpb "github.com/NpoolPlatform/message/npool/review/mgr/v2"
	reviewcli "github.com/NpoolPlatform/review-middleware/pkg/client/review"

	kycmgrpb "github.com/NpoolPlatform/message/npool/appuser/mgr/v2/kyc"

	constant "github.com/NpoolPlatform/ledger-gateway/pkg/message/const"

	signmethodpb "github.com/NpoolPlatform/message/npool/appuser/mgr/v2/signmethod"
	thirdmwcli "github.com/NpoolPlatform/third-middleware/pkg/client/verify"

	currencymwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"

	uuid1 "github.com/NpoolPlatform/go-service-framework/pkg/const/uuid"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"

	"github.com/google/uuid"
)

// nolint
func CreateWithdraw(
	ctx context.Context,
	appID, userID, coinTypeID, accountID string,
	amount decimal.Decimal,
	signMethod signmethodpb.SignMethodType,
	signAccount, verificationCode string,
) (
	*npool.Withdraw, error,
) {

	user, err := usermwcli.GetUser(ctx, appID, userID)
	if err != nil {
		return nil, err
	}
	if user.State != kycmgrpb.KycState_Approved {
		return nil, fmt.Errorf("permission denied")
	}

	if signMethod == signmethodpb.SignMethodType_Google {
		signAccount = user.GetGoogleSecret()
	}
	if err := thirdmwcli.VerifyCode(
		ctx,
		appID,
		signAccount,
		verificationCode,
		signMethod,
		usedfor.UsedFor_Withdraw,
	); err != nil {
		return nil, err
	}

	general, err := ledgermgrgeneralcli.GetGeneralOnly(ctx, &ledgermgrgeneralpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: coinTypeID,
		},
	})
	if err != nil {
		return nil, err
	}
	if general == nil {
		return nil, fmt.Errorf("insufficient funds")
	}

	account, err := useraccmwcli.GetAccountOnly(ctx, &useraccmwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: coinTypeID,
		},
		AccountID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: accountID,
		},
		Active: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: true,
		},
		Blocked: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: false,
		},
		UsedFor: &commonpb.Int32Val{
			Op:    cruder.EQ,
			Value: int32(accountmgrpb.AccountUsedFor_UserWithdraw),
		},
	})
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("invalid account")
	}

	coin, err := coininfocli.GetCoin(ctx, coinTypeID)
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid cointypeid")
	}
	if coin.Disabled {
		return nil, fmt.Errorf("invalid cointypeid")
	}

	appCoin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: coinTypeID,
		},
		Disabled: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: false,
		},
	})
	if err != nil {
		return nil, err
	}
	if appCoin == nil {
		return nil, fmt.Errorf("invalid app coin")
	}
	if appCoin.Disabled {
		return nil, fmt.Errorf("invalid app coin")
	}

	bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    coin.Name,
		Address: account.Address,
	})
	if err != nil {
		return nil, err
	}
	if bal == nil {
		return nil, fmt.Errorf("invalid account")
	}

	reviewTrigger := reviewpb.ReviewTriggerType_AutoReviewed

	hotacc, err := pltfaccmwcli.GetAccountOnly(ctx, &pltfaccmwpb.Conds{
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: coinTypeID,
		},
		UsedFor: &commonpb.Int32Val{
			Op:    cruder.EQ,
			Value: int32(accountmgrpb.AccountUsedFor_UserBenefitHot),
		},
		Active: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: true,
		},
		Backup: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: false,
		},
		Blocked: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: false,
		},
	})
	if err != nil {
		return nil, err
	}
	if hotacc == nil {
		return nil, fmt.Errorf("invalid hot wallet account")
	}

	bal, err = sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    coin.Name,
		Address: hotacc.Address,
	})
	if err != nil {
		return nil, err
	}
	if bal == nil {
		return nil, fmt.Errorf("invalid balance")
	}

	balance := decimal.RequireFromString(bal.BalanceStr)
	if balance.Cmp(amount) <= 0 {
		reviewTrigger = reviewpb.ReviewTriggerType_InsufficientFunds
	}

	if coin.ID != coin.FeeCoinTypeID {
		feecoin, err := coininfocli.GetCoin(ctx, coin.FeeCoinTypeID)
		if err != nil {
			return nil, err
		}
		if feecoin == nil {
			return nil, fmt.Errorf("invalid fee coin")
		}

		bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
			Name:    feecoin.Name,
			Address: hotacc.Address,
		})
		if err != nil {
			return nil, err
		}
		if bal == nil {
			return nil, fmt.Errorf("invalid balance")
		}

		feeAmount, err := decimal.NewFromString(feecoin.LowFeeAmount)
		if err != nil {
			return nil, err
		}

		balance := decimal.RequireFromString(bal.BalanceStr)
		if balance.Cmp(feeAmount) <= 0 {
			switch reviewTrigger {
			case reviewpb.ReviewTriggerType_InsufficientFunds:
				reviewTrigger = reviewpb.ReviewTriggerType_InsufficientFundsGas
			case reviewpb.ReviewTriggerType_AutoReviewed:
				reviewTrigger = reviewpb.ReviewTriggerType_InsufficientGas
			}
		}
	}

	threshold, err := decimal.NewFromString(appCoin.WithdrawAutoReviewAmount)
	if err != nil {
		return nil, err
	}

	if amount.Cmp(threshold) > 0 && reviewTrigger == reviewpb.ReviewTriggerType_AutoReviewed {
		reviewTrigger = reviewpb.ReviewTriggerType_LargeAmount
	}

	feeAmount, err := decimal.NewFromString(appCoin.WithdrawFeeAmount)
	if err != nil {
		return nil, err
	}
	if feeAmount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid fee amount")
	}

	if appCoin.WithdrawFeeByStableUSD {
		curr, err := currencymwcli.GetCoinCurrency(ctx, coin.ID)
		if err != nil {
			return nil, err
		}
		value, err := decimal.NewFromString(curr.MarketValueLow)
		if err != nil {
			return nil, err
		}
		if value.Cmp(decimal.NewFromInt(0)) <= 0 {
			return nil, fmt.Errorf("invalid coin price")
		}
		feeAmount = feeAmount.Div(value)
	}

	if amount.Cmp(feeAmount) <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}

	spendable, err := decimal.NewFromString(general.Spendable)
	if err != nil {
		return nil, err
	}
	if spendable.Cmp(amount) < 0 {
		return nil, fmt.Errorf("insufficient funds")
	}

	amountS := amount.String()
	feeAmountS := feeAmount.String()

	// TODO: move to TX
	// TODO: unlock if we fail before transaction created

	if err := ledgermwcli.LockBalance(
		ctx,
		appID, userID, coinTypeID, amount,
	); err != nil {
		return nil, err
	}

	needUnlock := true
	defer func() {
		if err == nil {
			return
		}
		if !needUnlock {
			return
		}
		_ = ledgermwcli.UnlockBalance(
			ctx,
			appID, userID, coinTypeID,
			ledgermgrdetailpb.IOSubType_Withdrawal,
			amount, decimal.NewFromInt(0),
			fmt.Sprintf(
				`{"AccountID":"%v","Timestamp":"%v"}`,
				accountID, time.Now(),
			),
		)
	}()

	// TODO: move to dtm to ensure data integrity
	// Create withdraw
	info, err := ledgermgrwithdrawcli.CreateWithdraw(ctx, &ledgermgrwithdrawpb.WithdrawReq{
		AppID:      &appID,
		UserID:     &userID,
		CoinTypeID: &coinTypeID,
		AccountID:  &accountID,
		Address:    &account.Address,
		Amount:     &amountS,
	})
	if err != nil {
		return nil, err
	}

	serviceName := constant.ServiceName
	objectType := reviewpb.ReviewObjectType_ObjectWithdrawal

	rv, err := reviewcli.CreateReview(ctx, &reviewpb.ReviewReq{
		AppID:      &appID,
		Domain:     &serviceName,
		ObjectType: &objectType,
		ObjectID:   &info.ID,
		Trigger:    &reviewTrigger,
	})
	if err != nil {
		return nil, err
	}
	if rv == nil {
		return nil, fmt.Errorf("invalid review")
	}

	if reviewTrigger == reviewpb.ReviewTriggerType_AutoReviewed {
		rstate := reviewpb.ReviewState_Approved
		reviewer := uuid1.InvalidUUIDStr

		if _, err := reviewcli.UpdateReview(ctx, &reviewpb.ReviewReq{
			ID:         &rv.ID,
			ReviewerID: &reviewer,
			State:      &rstate,
		}); err != nil {
			return nil, err
		}

		message := fmt.Sprintf(
			`{"AppID":"%v","UserID":"%v","Address":"%v","CoinName":"%v","WithdrawID":"%v"}`,
			appID,
			userID,
			account.Address,
			coin.Name,
			info.ID,
		)

		txType := txmgrpb.TxType_TxWithdraw

		// TODO: should be in dtm
		tx, err := txmwcli.CreateTx(ctx, &txmgrpb.TxReq{
			CoinTypeID:    &coinTypeID,
			FromAccountID: &hotacc.AccountID,
			ToAccountID:   &account.AccountID,
			Amount:        &amountS,
			FeeAmount:     &feeAmountS,
			Extra:         &message,
			Type:          &txType,
		})
		if err != nil {
			return nil, err
		}

		state := ledgermgrwithdrawpb.WithdrawState_Transferring
		if _, err := ledgermgrwithdrawcli.UpdateWithdraw(ctx, &ledgermgrwithdrawpb.WithdrawReq{
			ID:                    &info.ID,
			PlatformTransactionID: &tx.ID,
			State:                 &state,
		}); err != nil {
			needUnlock = false
			return nil, err
		}
	}

	needUnlock = false
	// Get withdraw
	return GetWithdraw(ctx, info.ID)
}

func GetWithdraw(ctx context.Context, id string) (*npool.Withdraw, error) {
	info, err := ledgermgrwithdrawcli.GetWithdraw(ctx, id)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("invalid withdraw")
	}

	coin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: info.AppID,
		},
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: info.CoinTypeID,
		},
	})
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid coin")
	}

	account, err := useraccmwcli.GetAccountOnly(ctx, &useraccmwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: info.AppID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: info.UserID,
		},
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: info.CoinTypeID,
		},
		AccountID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: info.AccountID,
		},
		Active: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: true,
		},
		Blocked: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: false,
		},
		UsedFor: &commonpb.Int32Val{
			Op:    cruder.EQ,
			Value: int32(accountmgrpb.AccountUsedFor_UserWithdraw),
		},
	})
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("invalid account")
	}

	message := ""

	// TODO: move to review middleware
	if info.State == ledgermgrwithdrawpb.WithdrawState_Rejected {
		rv, err := reviewcli.GetObjectReview(
			ctx,
			info.AppID,
			constant.ServiceName,
			id,
			reviewpb.ReviewObjectType_ObjectWithdrawal,
		)
		if err != nil {
			return nil, err
		}
		if rv == nil {
			return nil, fmt.Errorf("invalid review")
		}

		message = rv.Message
	}

	return &npool.Withdraw{
		CoinTypeID:    info.CoinTypeID,
		CoinName:      coin.Name,
		DisplayNames:  coin.DisplayNames,
		CoinLogo:      coin.Logo,
		CoinUnit:      coin.Unit,
		Amount:        info.Amount,
		CreatedAt:     info.CreatedAt,
		Address:       account.Address,
		AddressLabels: account.Labels,
		State:         info.State,
		Message:       message,
	}, nil
}

func GetWithdraws(
	ctx context.Context, appID, userID string, offset, limit int32,
) (
	[]*npool.Withdraw, uint32, error,
) {
	conds := &ledgermgrwithdrawpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
	}

	infos, total, err := ledgermgrwithdrawcli.GetWithdraws(ctx, conds, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(infos) == 0 {
		return []*npool.Withdraw{}, 0, nil
	}

	ids := []string{}
	for _, info := range infos {
		ids = append(ids, info.AccountID)
	}

	accounts, _, err := useraccmwcli.GetAccounts(ctx, &useraccmwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
		UsedFor: &commonpb.Int32Val{
			Op:    cruder.EQ,
			Value: int32(accountmgrpb.AccountUsedFor_UserWithdraw),
		},
		AccountIDs: &commonpb.StringSliceVal{
			Op:    cruder.IN,
			Value: ids,
		},
	}, 0, int32(len(ids)))
	if err != nil {
		return nil, 0, err
	}

	waccMap := map[string]*useraccmwpb.Account{}
	for _, acc := range accounts {
		waccMap[acc.AccountID] = acc
	}

	withdraws, err := expand(ctx, appID, infos, waccMap)
	if err != nil {
		return nil, 0, err
	}

	return withdraws, total, nil
}

func GetAppWithdraws(
	ctx context.Context, appID string, offset, limit int32,
) (
	[]*npool.Withdraw, uint32, error,
) {
	conds := &ledgermgrwithdrawpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
	}

	infos, total, err := ledgermgrwithdrawcli.GetWithdraws(ctx, conds, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(infos) == 0 {
		return []*npool.Withdraw{}, 0, nil
	}

	ids := []string{}
	for _, info := range infos {
		ids = append(ids, info.AccountID)
	}

	accounts, _, err := useraccmwcli.GetAccounts(ctx, &useraccmwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UsedFor: &commonpb.Int32Val{
			Op:    cruder.EQ,
			Value: int32(accountmgrpb.AccountUsedFor_UserWithdraw),
		},
		AccountIDs: &commonpb.StringSliceVal{
			Op:    cruder.IN,
			Value: ids,
		},
	}, 0, int32(len(ids)))
	if err != nil {
		return nil, 0, err
	}

	waccMap := map[string]*useraccmwpb.Account{}
	for _, acc := range accounts {
		waccMap[acc.AccountID] = acc
	}

	withdraws, err := expand(ctx, appID, infos, waccMap)
	if err != nil {
		return nil, 0, err
	}

	return withdraws, total, nil
}

func expand(
	ctx context.Context,
	appID string,
	infos []*ledgermgrwithdrawpb.Withdraw,
	waccMap map[string]*useraccmwpb.Account,
) (
	[]*npool.Withdraw, error,
) {
	ids := []string{}

	for _, info := range infos {
		if _, err := uuid.Parse(info.CoinTypeID); err != nil {
			continue
		}
		ids = append(ids, info.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		CoinTypeIDs: &commonpb.StringSliceVal{
			Op:    cruder.IN,
			Value: ids,
		},
	}, 0, int32(len(ids)))
	if err != nil {
		return nil, err
	}

	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.ID] = coin
	}

	wids := []string{}
	for _, w := range infos {
		wids = append(wids, w.ID)
	}

	reviews, err := reviewcli.GetObjectReviews(
		ctx,
		appID,
		constant.ServiceName,
		wids,
		reviewpb.ReviewObjectType_ObjectWithdrawal,
	)
	if err != nil {
		return nil, err
	}

	messageMap := map[string]string{}
	for _, r := range reviews {
		if r.State == reviewpb.ReviewState_Rejected {
			messageMap[r.ObjectID] = r.Message
		}
	}

	withdraws := []*npool.Withdraw{}
	for _, info := range infos {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			continue
		}

		address := info.Address
		labels := []string{}

		wacc, ok := waccMap[info.AccountID]
		if ok {
			labels = wacc.Labels
			address = wacc.Address
		}

		withdraws = append(withdraws, &npool.Withdraw{
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

	return withdraws, nil
}

func GetIntervalWithdraws(
	ctx context.Context, appID, userID string, start, end uint32, offset, limit int32,
) (
	[]*npool.Withdraw, uint32, error,
) {
	return nil, 0, fmt.Errorf("NOT IMPLEMENTED")
}
