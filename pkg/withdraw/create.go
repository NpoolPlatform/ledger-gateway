package withdraw

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	txnotifmwpb "github.com/NpoolPlatform/message/npool/notif/mw/v1/notif/tx"
	txnotifcli "github.com/NpoolPlatform/notif-middleware/pkg/client/notif/tx"

	notifmwpb "github.com/NpoolPlatform/message/npool/notif/mw/v1/notif"
	tmplmwpb "github.com/NpoolPlatform/message/npool/notif/mw/v1/template"
	notifmwcli "github.com/NpoolPlatform/notif-middleware/pkg/client/notif"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"

	"github.com/shopspring/decimal"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"
	withdrawmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/withdraw"
	withdrawmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw"

	lockcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"

	txmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/tx"
	txmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/tx"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"

	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	useraccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/user"
	useraccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/user"

	pltfaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/platform"
	pltfaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/platform"

	reviewpb "github.com/NpoolPlatform/message/npool/review/mw/v2/review"
	reviewcli "github.com/NpoolPlatform/review-middleware/pkg/client/review"

	constant "github.com/NpoolPlatform/ledger-gateway/pkg/message/const"

	usercodemwcli "github.com/NpoolPlatform/basal-middleware/pkg/client/usercode"
	usercodemwpb "github.com/NpoolPlatform/message/npool/basal/mw/v1/usercode"

	currencymwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	currencymwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"

	uuid1 "github.com/NpoolPlatform/go-service-framework/pkg/const/uuid"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	ledgerpb "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

// nolint
func (h *Handler) CreateWithdraw(ctx context.Context) (*npool.Withdraw, error) {
	user, _ := usermwcli.GetUser(ctx, *h.AppID, *h.UserID)
	if user.State != basetypes.KycState_Approved {
		return nil, fmt.Errorf("kyc not approved, user id(%v)", h.UserID)
	}
	if *h.AccountType == basetypes.SignMethod_Google {
		h.Account = &user.GoogleSecret
	}

	if err := usercodemwcli.VerifyUserCode(ctx, &usercodemwpb.VerifyUserCodeRequest{
		Prefix:      basetypes.Prefix_PrefixUserCode.String(),
		AppID:       *h.AppID,
		Account:     *h.Account,
		AccountType: *h.AccountType,
		UsedFor:     basetypes.UsedFor_Withdraw,
		Code:        *h.VerificationCode,
	}); err != nil {
		return nil, err
	}

	ledger, err := ledgermwcli.GetLedgerOnly(ctx, &ledgermwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
	})
	if err != nil {
		return nil, err
	}
	if ledger == nil {
		return nil, fmt.Errorf("ledger not exist, appid(%v), userid(%v), cointypeid(%v)", *h.AppID, *h.UserID, *h.CoinTypeID)
	}

	spendable, err := decimal.NewFromString(ledger.Spendable)
	if err != nil {
		return nil, err
	}
	if spendable.Cmp(*h.Amount) < 0 {
		return nil, fmt.Errorf("insufficient funds, spendable(%v)", spendable.String())
	}

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
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("could not find active account for withdraw, cointypeid(%v)", *h.CoinTypeID)
	}

	coin, _ := coininfocli.GetCoin(ctx, *h.CoinTypeID)
	if coin.Disabled {
		return nil, fmt.Errorf("coin disabled")
	}
	appCoin, err := appcoinmwcli.GetCoin(ctx, *h.CoinTypeID)
	if err != nil {
		return nil, err
	}
	if appCoin.Disabled {
		return nil, fmt.Errorf("app coin disabled")
	}

	maxWithdrawAmount, err := decimal.NewFromString(appCoin.MaxAmountPerWithdraw)
	if err != nil {
		return nil, err
	}
	if h.Amount.Cmp(maxWithdrawAmount) > 0 {
		return nil, fmt.Errorf("amount greater than max amount per withdraw %v", maxWithdrawAmount.String())
	}

	if !strings.Contains(coin.Name, "ironfish") {
		bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
			Name:    coin.Name,
			Address: account.Address,
		})
		if err != nil {
			return nil, err
		}
		if bal == nil {
			return nil, fmt.Errorf("coin get balance fail, name(%v), address(%v)", coin.Name, account.Address)
		}
	}

	hotacc, err := pltfaccmwcli.GetAccountOnly(ctx, &pltfaccmwpb.Conds{
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
		UsedFor:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(basetypes.AccountUsedFor_UserBenefitHot)},
		Active:     &basetypes.BoolVal{Op: cruder.EQ, Value: true},
		Backup:     &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		Blocked:    &basetypes.BoolVal{Op: cruder.EQ, Value: false},
	})
	if err != nil {
		return nil, err
	}
	if hotacc == nil {
		return nil, fmt.Errorf("invalid hot wallet account")
	}

	bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    coin.Name,
		Address: hotacc.Address,
	})
	if err != nil {
		return nil, err
	}
	if bal == nil {
		return nil, fmt.Errorf("platform account get balance fail, name(%v), hot address(%v)", coin.Name, hotacc.Address)
	}

	reviewTrigger := reviewpb.ReviewTriggerType_AutoReviewed

	balance := decimal.RequireFromString(bal.BalanceStr)
	if balance.Cmp(*h.Amount) <= 0 {
		reviewTrigger = reviewpb.ReviewTriggerType_InsufficientFunds
	}

	if coin.ID != coin.FeeCoinTypeID {
		feecoin, err := coininfocli.GetCoin(ctx, coin.FeeCoinTypeID)
		if err != nil {
			return nil, err
		}
		if feecoin == nil {
			return nil, fmt.Errorf("invalid fee coin, cointypeid(%v)", coin.FeeCoinTypeID)
		}

		bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
			Name:    feecoin.Name,
			Address: hotacc.Address,
		})
		if err != nil {
			return nil, err
		}
		if bal == nil {
			return nil, fmt.Errorf("fee coin get balance fail")
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

	if h.Amount.Cmp(threshold) > 0 && reviewTrigger == reviewpb.ReviewTriggerType_AutoReviewed {
		reviewTrigger = reviewpb.ReviewTriggerType_LargeAmount
	}

	feeAmount, err := decimal.NewFromString(appCoin.WithdrawFeeAmount)
	if err != nil {
		return nil, err
	}
	if feeAmount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid withdraw fee amount")
	}

	if appCoin.WithdrawFeeByStableUSD {
		curr, err := currencymwcli.GetCurrencyOnly(ctx, &currencymwpb.Conds{
			CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: coin.ID},
		})
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

	if h.Amount.Cmp(feeAmount) <= 0 {
		return nil, fmt.Errorf("amount(%v) less than fee amount(%v)", h.Amount.String(), feeAmount.String())
	}

	amountStr := h.Amount.String()
	feeAmountStr := feeAmount.String()
	outcoming := "0"
	// TODO: move to TX
	// TODO: unlock if we fail before transaction created

	if _, err := lockcli.AddBalance(ctx, &ledgermwpb.LedgerReq{
		AppID:      h.AppID,
		UserID:     h.UserID,
		CoinTypeID: h.CoinTypeID,
		Locked:     &amountStr,
		Outcoming:  &outcoming,
	}); err != nil {
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

		ioSubType := ledgerpb.IOSubType_Withdrawal
		extra := fmt.Sprintf(`{"AccountID":"%v","Timestamp":"%v"}`, *h.AccountID, time.Now())
		_, err := lockcli.SubBalance(ctx, &ledgermwpb.LedgerReq{
			AppID:      h.AppID,
			UserID:     h.UserID,
			CoinTypeID: h.CoinTypeID,
			IOSubType:  &ioSubType,
			IOExtra:    &extra,
			Locked:     &amountStr,
			Outcoming:  &outcoming,
		})
		if err != nil {
			logger.Sugar().Error("SubBalance failed, err %v", err)
		}
	}()

	// TODO: move to dtm to ensure data integrity
	// Create withdraw
	info, err := withdrawmwcli.CreateWithdraw(ctx, &withdrawmwpb.WithdrawReq{
		AppID:      h.AppID,
		UserID:     h.UserID,
		CoinTypeID: h.CoinTypeID,
		AccountID:  h.AccountID,
		Address:    &account.Address,
		Amount:     &amountStr,
	})
	if err != nil {
		return nil, err
	}

	serviceName := constant.ServiceName
	objectType := reviewpb.ReviewObjectType_ObjectWithdrawal

	rv, err := reviewcli.CreateReview(ctx, &reviewpb.ReviewReq{
		AppID:      h.AppID,
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
			*h.AppID,
			*h.UserID,
			account.Address,
			coin.Name,
			info.ID,
		)

		txType := basetypes.TxType_TxWithdraw

		// TODO: should be in dtm
		tx, err := txmwcli.CreateTx(ctx, &txmwpb.TxReq{
			CoinTypeID:    h.CoinTypeID,
			FromAccountID: &hotacc.AccountID,
			ToAccountID:   &account.AccountID,
			Amount:        &amountStr,
			FeeAmount:     &feeAmountStr,
			Extra:         &message,
			Type:          &txType,
		})
		if err != nil {
			return nil, err
		}

		state := ledgerpb.WithdrawState_Transferring
		if _, err := withdrawmwcli.UpdateWithdraw(ctx, &withdrawmwpb.WithdrawReq{
			ID:                    &info.ID,
			PlatformTransactionID: &tx.ID,
			State:                 &state,
		}); err != nil {
			needUnlock = false
			return nil, err
		}

		txNotifState := txnotifmwpb.TxState_WaitSuccess
		txNotifType := basetypes.TxType_TxWithdraw
		logger.Sugar().Errorw(
			"CreateTx",
			"txNotifState", txNotifState,
			"txNotifType", txNotifType,
		)
		_, err = txnotifcli.CreateTx(ctx, &txnotifmwpb.TxReq{
			TxID:       &tx.ID,
			NotifState: &txNotifState,
			TxType:     &txNotifType,
		})
		if err != nil {
			logger.Sugar().Errorw("CreateTx", "Error", err)
		}
	}

	needUnlock = false
	now := uint32(time.Now().Unix())

	_, err = notifmwcli.GenerateNotifs(ctx, &notifmwpb.GenerateNotifsRequest{
		AppID:     *h.AppID,
		UserID:    *h.UserID,
		EventType: basetypes.UsedFor_WithdrawalRequest,
		NotifType: basetypes.NotifType_NotifUnicast,
		Vars: &tmplmwpb.TemplateVars{
			Username:  &user.Username,
			Amount:    &amountStr,
			CoinUnit:  &coin.Unit,
			Address:   &account.Address,
			Timestamp: &now,
		},
	})
	if err != nil {
		logger.Sugar().Errorw("CreateTx", "Error", err)
	}

	infos, _, err := h.GetWithdraws(ctx)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("cannot find withdraw")
	}
	if len(infos) > 1 {
		return nil, fmt.Errorf("to many ")
	}
	return infos[0], nil
}
