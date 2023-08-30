package withdraw

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/google/uuid"

	txnotifmwpb "github.com/NpoolPlatform/message/npool/notif/mw/v1/notif/tx"
	txnotifcli "github.com/NpoolPlatform/notif-middleware/pkg/client/notif/tx"

	notifmwpb "github.com/NpoolPlatform/message/npool/notif/mw/v1/notif"
	tmplmwpb "github.com/NpoolPlatform/message/npool/notif/mw/v1/template"
	notifmwcli "github.com/NpoolPlatform/notif-middleware/pkg/client/notif"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"

	"github.com/shopspring/decimal"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"
	withdrawmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/withdraw"
	withdrawmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw"

	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	coinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin"

	txmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/tx"
	txmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/tx"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	useraccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/user"
	useraccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/user"

	pltfaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/platform"
	pltfaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/platform"

	reviewpb "github.com/NpoolPlatform/message/npool/review/mw/v2/review"
	reviewcli "github.com/NpoolPlatform/review-middleware/pkg/client/review"

	constant "github.com/NpoolPlatform/ledger-gateway/pkg/servicename"

	usercodemwcli "github.com/NpoolPlatform/basal-middleware/pkg/client/usercode"
	usercodemwpb "github.com/NpoolPlatform/message/npool/basal/mw/v1/usercode"

	currencymwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	currencymwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"

	uuid1 "github.com/NpoolPlatform/go-service-framework/pkg/const/uuid"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	ledgerpb "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	"github.com/NpoolPlatform/go-service-framework/pkg/pubsub"
	ledgermwsvcname "github.com/NpoolPlatform/ledger-middleware/pkg/servicename"
	reviewmwsvcname "github.com/NpoolPlatform/review-middleware/pkg/servicename"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"
)

type createHandler struct {
	*Handler
	RequestTimeoutSeconds  int64
	user                   *usermwpb.User
	account                *useraccmwpb.Account
	accountBalance         decimal.Decimal
	platformAccount        *pltfaccmwpb.Account
	platformAccountBalance decimal.Decimal
	coin                   *coinmwpb.Coin
	feecoin                *coinmwpb.Coin
	appcoin                *appcoinmwpb.Coin
	feeBalance             decimal.Decimal
	reviewTrigger          reviewpb.ReviewTriggerType
	reviewID               *string
	txID                   *string
}

func (h *createHandler) verifyUserCode(ctx context.Context) error {
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

	if err := usercodemwcli.VerifyUserCode(ctx, &usercodemwpb.VerifyUserCodeRequest{
		Prefix:      basetypes.Prefix_PrefixUserCode.String(),
		AppID:       *h.AppID,
		Account:     *h.Account,
		AccountType: *h.AccountType,
		UsedFor:     basetypes.UsedFor_Withdraw,
		Code:        *h.VerificationCode,
	}); err != nil {
		return err
	}
	return nil
}

func (h *createHandler) checkBalance(ctx context.Context) error {
	ledger, err := ledgermwcli.GetLedgerOnly(ctx, &ledgermwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
	})
	if err != nil {
		return err
	}
	if ledger == nil {
		return fmt.Errorf("ledger not exist, appid(%v), userid(%v), cointypeid(%v)", *h.AppID, *h.UserID, *h.CoinTypeID)
	}
	spendable, err := decimal.NewFromString(ledger.Spendable)
	if err != nil {
		return err
	}
	if spendable.Cmp(*h.Amount) < 0 {
		return fmt.Errorf("insufficient funds, spendable(%v)", spendable.String())
	}
	maxAmount, err := decimal.NewFromString(h.appcoin.MaxAmountPerWithdraw)
	if err != nil {
		return err
	}
	if h.Amount.Cmp(maxAmount) > 0 {
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
		return fmt.Errorf("could not find active account for withdraw, cointypeid(%v)", *h.CoinTypeID)
	}
	h.account = account

	if !strings.Contains(h.coin.Name, "ironfish") {
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
		h.accountBalance = decimal.RequireFromString(bal.BalanceStr)
	}
	return nil
}

func (h *createHandler) getPlatformAccount(ctx context.Context) error {
	hotacc, err := pltfaccmwcli.GetAccountOnly(ctx, &pltfaccmwpb.Conds{
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
		UsedFor:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(basetypes.AccountUsedFor_UserBenefitHot)},
		Active:     &basetypes.BoolVal{Op: cruder.EQ, Value: true},
		Backup:     &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		Blocked:    &basetypes.BoolVal{Op: cruder.EQ, Value: false},
	})
	if err != nil {
		return err
	}
	if hotacc == nil {
		return fmt.Errorf("invalid hot wallet account")
	}

	bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    h.coin.Name,
		Address: hotacc.Address,
	})
	if err != nil {
		return err
	}
	if bal == nil {
		return fmt.Errorf("can not get balance")
	}
	h.platformAccountBalance = decimal.RequireFromString(bal.BalanceStr)
	return nil
}

func (h *createHandler) getFeeCoinBalance(ctx context.Context) error {
	if h.coin.ID != h.coin.FeeCoinTypeID {
		feecoin, err := coininfocli.GetCoin(ctx, h.coin.FeeCoinTypeID)
		if err != nil {
			return err
		}
		if feecoin == nil {
			return fmt.Errorf("invalid fee coin")
		}
		h.feecoin = feecoin

		bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
			Name:    feecoin.Name,
			Address: h.platformAccount.Address,
		})
		if err != nil {
			return err
		}
		if bal == nil {
			return fmt.Errorf("invalid balance")
		}
		h.feeBalance = decimal.RequireFromString(bal.BalanceStr)
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

func (h *createHandler) getWithdrawFeeAmount(ctx context.Context) (string, error) {
	feeAmount, err := decimal.NewFromString(h.appcoin.WithdrawFeeAmount)
	if err != nil {
		return "", err
	}
	if feeAmount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return "", fmt.Errorf("invalid fee amount")
	}

	if h.appcoin.WithdrawFeeByStableUSD {
		curr, err := currencymwcli.GetCurrencyOnly(ctx, &currencymwpb.Conds{
			CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: h.coin.ID},
		})
		if err != nil {
			return "", err
		}
		value, err := decimal.NewFromString(curr.MarketValueLow)
		if err != nil {
			return "", err
		}
		if value.Cmp(decimal.NewFromInt(0)) <= 0 {
			return "", fmt.Errorf("invalid coin price")
		}
		feeAmount = feeAmount.Div(value)
	}
	if h.Amount.Cmp(feeAmount) <= 0 {
		return "", fmt.Errorf("amount(%v) less than fee amount(%v)", h.Amount.String(), feeAmount.String())
	}
	return feeAmount.String(), nil
}

func (h *createHandler) setReviewTrigger() error {
	reviewTrigger := reviewpb.ReviewTriggerType_AutoReviewed
	if h.platformAccountBalance.Cmp(*h.Amount) <= 0 {
		reviewTrigger = reviewpb.ReviewTriggerType_InsufficientFunds
	}
	if h.feecoin != nil {
		feeAmount, err := decimal.NewFromString(h.feecoin.LowFeeAmount)
		if err != nil {
			return err
		}
		if h.feeBalance.Cmp(feeAmount) <= 0 {
			switch reviewTrigger {
			case reviewpb.ReviewTriggerType_InsufficientFunds:
				reviewTrigger = reviewpb.ReviewTriggerType_InsufficientFundsGas
			case reviewpb.ReviewTriggerType_AutoReviewed:
				reviewTrigger = reviewpb.ReviewTriggerType_InsufficientGas
			}
		}
	}
	thresold, err := decimal.NewFromString(h.appcoin.WithdrawAutoReviewAmount)
	if err != nil {
		return err
	}
	if h.Amount.Cmp(thresold) > 0 && reviewTrigger == reviewpb.ReviewTriggerType_AutoReviewed {
		reviewTrigger = reviewpb.ReviewTriggerType_LargeAmount
	}
	h.reviewTrigger = reviewTrigger
	return nil
}

func (h *createHandler) withCreateWithdraw(dispose *dtmcli.SagaDispose) {
	amount := h.Amount.String()
	req := &withdrawmwpb.WithdrawReq{
		ID:         h.ID,
		AppID:      h.AppID,
		UserID:     h.UserID,
		CoinTypeID: h.CoinTypeID,
		AccountID:  h.AccountID,
		Address:    &h.account.Address,
		Amount:     &amount,
	}
	if h.reviewTrigger == reviewpb.ReviewTriggerType_AutoReviewed {
		req.PlatformTransactionID = h.txID
	}

	dispose.Add(
		ledgermwsvcname.ServiceDomain,
		"ledger.middleware.withdraw.v2.Middleware/CreateWithdraw",
		"ledger.middleware.withdraw.v2.Middleware/DeleteWithdraw",
		&withdrawmwpb.CreateWithdrawRequest{
			Info: req,
		},
	)
}

func (h *createHandler) withCreateReview(dispose *dtmcli.SagaDispose) {
	domain := constant.ServiceDomain
	objectType := reviewpb.ReviewObjectType_ObjectWithdrawal

	req := &reviewpb.ReviewReq{
		ID:         h.reviewID,
		AppID:      h.AppID,
		Domain:     &domain,
		ObjectType: &objectType,
		ObjectID:   h.ID,
		Trigger:    &h.reviewTrigger,
	}
	dispose.Add(
		reviewmwsvcname.ServiceDomain,
		"review.middleware.review.v2.Middleware/CreateReview",
		"review.middleware.review.v2.Middleware/DeleteReview",
		&reviewpb.CreateReviewRequest{
			Info: req,
		},
	)
}

func (h *createHandler) notifyCreateTx(req *txmwpb.TxReq) {
	if err := pubsub.WithPublisher(func(publisher *pubsub.Publisher) error {
		return publisher.Update(
			basetypes.MsgID_CreateTxReq.String(),
			nil,
			nil,
			nil,
			req,
		)
	}); err != nil {
		logger.Sugar().Errorw(
			"notifyCreateTx",
			"ID", *req.ID,
			"CoinTypeID", *req.CoinTypeID,
			"FromAccountID", *req.FromAccountID,
			"ToAccountID", *req.ToAccountID,
			"Amount", *req.Amount,
			"FeeAmount", *req.FeeAmount,
			"Extra", *req.Extra,
			"Type", *req.Type,
			"Error", err,
		)
	}
}

// nolint
func (h *Handler) CreateWithdraw(ctx context.Context) (*npool.Withdraw, error) {
	handler := &createHandler{
		Handler:               h,
		reviewTrigger:         reviewpb.ReviewTriggerType_AutoReviewed,
		RequestTimeoutSeconds: 10,
	}
	if err := handler.verifyUserCode(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkCoin(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkBalance(ctx); err != nil {
		return nil, err
	}
	if err := handler.getUserAccount(ctx); err != nil {
		return nil, err
	}
	if err := handler.getPlatformAccount(ctx); err != nil {
		return nil, err
	}
	if err := handler.getFeeCoinBalance(ctx); err != nil {
		return nil, err
	}
	if err := handler.setReviewTrigger(); err != nil {
		return nil, err
	}
	feeAmount, err := handler.getWithdrawFeeAmount(ctx)
	if err != nil {
		return nil, err
	}

	id := uuid.NewString()
	if h.ID == nil {
		h.ID = &id
	}

	reviewID := uuid.NewString()
	if handler.reviewID == nil {
		handler.reviewID = &reviewID
	}

	txID := uuid.NewString()
	if handler.txID == nil {
		handler.txID = &txID
	}

	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: handler.RequestTimeoutSeconds,
	})

	handler.withCreateWithdraw(sagaDispose)
	handler.withCreateReview(sagaDispose)
	if err := dtmcli.WithSaga(ctx, sagaDispose); err != nil {
		return nil, err
	}

	if handler.reviewTrigger == reviewpb.ReviewTriggerType_AutoReviewed {
		rstate := reviewpb.ReviewState_Approved
		reviewer := uuid1.InvalidUUIDStr

		if _, err := reviewcli.UpdateReview(ctx, &reviewpb.ReviewReq{
			ID:         &reviewID,
			ReviewerID: &reviewer,
			State:      &rstate,
		}); err != nil {
			return nil, err
		}

		message := fmt.Sprintf(
			`{"AppID":"%v","UserID":"%v","Address":"%v","CoinName":"%v","WithdrawID":"%v"}`,
			*h.AppID,
			*h.UserID,
			handler.account.Address,
			handler.coin.Name,
			*h.ID,
		)
		txType := basetypes.TxType_TxWithdraw
		amountStr := h.Amount.String()
		req := &txmwpb.TxReq{
			ID:            handler.txID,
			CoinTypeID:    h.CoinTypeID,
			FromAccountID: &handler.platformAccount.AccountID,
			ToAccountID:   &handler.account.AccountID,
			Amount:        &amountStr,
			FeeAmount:     &feeAmount,
			Extra:         &message,
			Type:          &txType,
		}
		tx, err := txmwcli.CreateTx(ctx, req)
		if err != nil {
			handler.notifyCreateTx(req)
			return nil, err
		}

		state := ledgerpb.WithdrawState_Transferring
		if _, err := withdrawmwcli.UpdateWithdraw(ctx, &withdrawmwpb.WithdrawReq{
			ID:                    h.ID,
			PlatformTransactionID: &tx.ID,
			State:                 &state,
		}); err != nil {
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

	now := uint32(time.Now().Unix())
	amountStr := h.Amount.String()
	_, err = notifmwcli.GenerateNotifs(ctx, &notifmwpb.GenerateNotifsRequest{
		AppID:     *h.AppID,
		UserID:    *h.UserID,
		EventType: basetypes.UsedFor_WithdrawalRequest,
		NotifType: basetypes.NotifType_NotifUnicast,
		Vars: &tmplmwpb.TemplateVars{
			Username:  &handler.user.Username,
			Amount:    &amountStr,
			CoinUnit:  &handler.coin.Unit,
			Address:   &handler.account.Address,
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
