package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermgrwithdrawcli "github.com/NpoolPlatform/ledger-manager/pkg/client/withdraw"
	ledgermgrwithdrawpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/withdraw"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	billingcli "github.com/NpoolPlatform/cloud-hashing-billing/pkg/client"
	billingpb "github.com/NpoolPlatform/message/npool/cloud-hashing-billing"

	reviewpb "github.com/NpoolPlatform/message/npool/review-service"
	reviewmgrpb "github.com/NpoolPlatform/message/npool/review/mgr/v2"
	reviewcli "github.com/NpoolPlatform/review-service/pkg/client"
	reviewconst "github.com/NpoolPlatform/review-service/pkg/const"

	constant "github.com/NpoolPlatform/ledger-gateway/pkg/message/const"

	currency "github.com/NpoolPlatform/oracle-manager/pkg/middleware/currency"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"

	"github.com/google/uuid"
)

const (
	defaultLimitAmount = 10000.0
	leastLimitAmount   = 0.001
)

func coinLimit(ctx context.Context, coin *coininfopb.CoinInfo, setting *billingpb.AppWithdrawSetting) (float64, error) {
	limit := defaultLimitAmount

	if setting != nil {
		// TODO: use decimal for amount
		limit = setting.WithdrawAutoReviewCoinAmount
	}

	if limit == 0 {
		psetting, err := billingcli.GetPlatformSetting(ctx)
		if err != nil {
			return defaultLimitAmount, err
		}

		price, err := currency.USDPrice(ctx, coin.Name)
		if err != nil {
			return defaultLimitAmount, err
		}

		limit = psetting.WithdrawAutoReviewUSDAmount / price
	}

	if limit < leastLimitAmount {
		return leastLimitAmount, nil
	}

	return limit, nil
}

// nolint
func CreateWithdraw(
	ctx context.Context,
	appID, userID, coinTypeID, accountID string,
	amount decimal.Decimal,
) (
	*npool.Withdraw, error,
) {
	// Try lock balance
	if err := ledgermwcli.LockBalance(
		ctx,
		appID, userID, coinTypeID, amount,
	); err != nil {
		return nil, err
	}

	// Check account
	account, err := billingcli.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("invalid account")
	}

	// Check account is belong to user and used for withdraw
	was, err := billingcli.GetWithdrawAccounts(ctx, appID, userID)
	if err != nil {
		return nil, err
	}
	found := false
	for _, wa := range was {
		if wa.AccountID == account.ID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("not user's withdraw address")
	}

	reviewTrigger := reviewmgrpb.ReviewTriggerType_AutoReviewed

	// Check hot wallet balance
	coin, err := coininfocli.GetCoinInfo(ctx, coinTypeID)
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid coin")
	}

	cs, err := billingcli.GetCoinSetting(ctx, coin.ID)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, fmt.Errorf("invalid coin setting")
	}

	hotacc, err := billingcli.GetAccount(ctx, cs.UserOnlineAccountID)
	if err != nil {
		return nil, err
	}
	if hotacc == nil {
		return nil, fmt.Errorf("invalid account")
	}

	bal, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    coin.Name,
		Address: hotacc.Address,
	})
	if err != nil {
		return nil, err
	}
	if bal == nil {
		return nil, fmt.Errorf("invalid balance")
	}

	// TODO: also check gas insufficient
	balance := decimal.RequireFromString(bal.BalanceStr)
	if balance.Cmp(amount) <= 0 {
		reviewTrigger = reviewmgrpb.ReviewTriggerType_InsufficientFunds
	}

	// Check auto review threshold
	ws, err := billingcli.GetWithdrawSetting(ctx, appID, coinTypeID)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, fmt.Errorf("invalid withdraw setting")
	}

	limit, err := coinLimit(ctx, coin, ws)
	if err != nil {
		return nil, err
	}

	threshold := decimal.NewFromFloat(limit)
	if amount.Cmp(threshold) > 0 && reviewTrigger == reviewmgrpb.ReviewTriggerType_AutoReviewed {
		reviewTrigger = reviewmgrpb.ReviewTriggerType_LargeAmount
	}

	price, err := currency.USDPrice(ctx, coin.Name)
	if err != nil {
		return nil, err
	}
	if price <= 0 {
		return nil, fmt.Errorf("invalid coin price")
	}

	const feeUSDAmount = 2
	feeAmount := feeUSDAmount / price

	amountS := amount.String()

	if amount.Cmp(decimal.NewFromFloat(feeAmount)) < 0 {
		return nil, fmt.Errorf("invalid amount")
	}

	// TODO: move to dtm to ensure data integrity
	// Create withdraw
	info, err := ledgermgrwithdrawcli.CreateWithdraw(ctx, &ledgermgrwithdrawpb.WithdrawReq{
		AppID:      &appID,
		UserID:     &userID,
		CoinTypeID: &coinTypeID,
		AccountID:  &accountID,
		Amount:     &amountS,
	})
	if err != nil {
		return nil, err
	}

	// Create review
	rv, err := reviewcli.CreateReview(ctx, &reviewpb.Review{
		AppID:      appID,
		Domain:     constant.ServiceName,
		ObjectType: reviewmgrpb.ReviewObjectType_ObjectWithdrawal.String(),
		ObjectID:   info.ID,
		State:      reviewmgrpb.ReviewState_Wait.String(),
		Trigger:    reviewTrigger.String(),
	})
	if err != nil {
		return nil, err
	}

	if reviewTrigger == reviewmgrpb.ReviewTriggerType_AutoReviewed {
		// Formatted to use formatted approved
		// rv.State = reviewmgrpb.ReviewState_Approved.String()
		rv.State = reviewconst.StateApproved
		if _, err := reviewcli.UpdateReview(ctx, rv); err != nil {
			return nil, err
		}

		// TODO: should be in dtm
		tx, err := billingcli.CreateTransaction(ctx, &billingpb.CoinAccountTransaction{
			AppID:          appID,
			UserID:         userID,
			CoinTypeID:     coinTypeID,
			GoodID:         uuid.UUID{}.String(),
			FromAddressID:  hotacc.ID,
			ToAddressID:    account.ID,
			Amount:         amount.InexactFloat64(),
			TransactionFee: feeAmount,
			Message:        fmt.Sprintf("user withdraw at %v", time.Now()),
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
			return nil, err
		}
	}

	// Get withdraw
	return GetWithdraw(ctx, info.ID)
}

// nolint
func GetWithdraw(ctx context.Context, id string) (*npool.Withdraw, error) {
	info, err := ledgermgrwithdrawcli.GetWithdraw(ctx, id)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("invalid withdraw")
	}

	coin, err := coininfocli.GetCoinInfo(ctx, info.CoinTypeID)
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid coin")
	}

	account, err := billingcli.GetAccount(ctx, info.AccountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("invalid account")
	}

	// TODO: also add account labels

	// TODO: move to review middleware
	reviews, err := reviewcli.GetObjectReviews(
		ctx,
		info.AppID, constant.ServiceName,
		reviewmgrpb.ReviewObjectType_ObjectWithdrawal.String(),
		info.ID,
	)
	if err != nil {
		return nil, err
	}

	state := ledgermgrwithdrawpb.WithdrawState_Reviewing
	waitTs := uint32(0)
	rejectedTs := uint32(0)
	message := ""

	for _, r := range reviews {
		switch r.State {
		case reviewconst.StateWait:
			fallthrough //nolint
		case reviewmgrpb.ReviewState_Wait.String():
			if waitTs < r.CreateAt {
				waitTs = r.CreateAt
			}
		case reviewconst.StateRejected:
			fallthrough //nolint
		case reviewmgrpb.ReviewState_Rejected.String():
			if rejectedTs < r.CreateAt {
				rejectedTs = r.CreateAt
				message = r.Message
			}
		}
	}

	if waitTs > rejectedTs {
		state = ledgermgrwithdrawpb.WithdrawState_Reviewing
		message = ""
	} else {
		state = ledgermgrwithdrawpb.WithdrawState_Rejected
	}

	for _, r := range reviews {
		switch r.State {
		case reviewconst.StateApproved:
			fallthrough //nolint
		case reviewmgrpb.ReviewState_Approved.String():
			state = ledgermgrwithdrawpb.WithdrawState_Successful
			message = ""
		}
	}

	return &npool.Withdraw{
		CoinTypeID:    info.CoinTypeID,
		CoinName:      coin.Name,
		CoinLogo:      coin.Logo,
		CoinUnit:      coin.Unit,
		Amount:        info.Amount,
		CreatedAt:     info.CreatedAt,
		Address:       account.Address,
		AddressLabels: "TODO: to be filled",
		State:         state, // TODO: get transactions for Transferring/TransactionFail state
		Message:       message,
	}, nil
}

// nolint
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

	coins, err := coininfocli.GetCoinInfos(ctx, cruder.NewFilterConds())
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*coininfopb.CoinInfo{}
	for _, coin := range coins {
		coinMap[coin.ID] = coin
	}

	accounts, err := billingcli.GetAccounts(ctx)
	if err != nil {
		return nil, 0, err
	}

	accMap := map[string]*billingpb.CoinAccountInfo{}
	for _, acc := range accounts {
		accMap[acc.ID] = acc
	}

	// TODO: also add account labels

	// TODO: move to review middleware
	reviews, err := reviewcli.GetDomainReviews(
		ctx,
		appID, constant.ServiceName, reviewmgrpb.ReviewObjectType_ObjectWithdrawal.String(),
	)
	if err != nil {
		return nil, 0, err
	}

	stateMap := map[string]ledgermgrwithdrawpb.WithdrawState{}
	waitTsMap := map[string]uint32{}
	rejectedTsMap := map[string]uint32{}

	messageMap := map[string]string{}

	for _, r := range reviews {
		switch r.State {
		case reviewconst.StateWait:
			fallthrough //nolint
		case reviewmgrpb.ReviewState_Wait.String():
			if waitTsMap[r.ObjectID] < r.CreateAt {
				waitTsMap[r.ObjectID] = r.CreateAt
			}
		case reviewconst.StateRejected:
			fallthrough //nolint
		case reviewmgrpb.ReviewState_Rejected.String():
			if rejectedTsMap[r.ObjectID] < r.CreateAt {
				rejectedTsMap[r.ObjectID] = r.CreateAt
				messageMap[r.ObjectID] = r.Message
			}
		}
	}

	for oid, waitTs := range waitTsMap {
		rejectedTs, ok := rejectedTsMap[oid]
		if !ok || waitTs > rejectedTs {
			stateMap[oid] = ledgermgrwithdrawpb.WithdrawState_Reviewing
			messageMap[oid] = ""
			continue
		}
		stateMap[oid] = ledgermgrwithdrawpb.WithdrawState_Rejected
	}

	for _, r := range reviews {
		switch r.State {
		case reviewconst.StateApproved:
			fallthrough //nolint
		case reviewmgrpb.ReviewState_Approved.String():
			stateMap[r.ObjectID] = ledgermgrwithdrawpb.WithdrawState_Successful
		}
	}

	withdraws := []*npool.Withdraw{}
	for _, info := range infos {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		acc, ok := accMap[info.AccountID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid account")
		}

		state, ok := stateMap[info.ID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid review state")
		}

		withdraws = append(withdraws, &npool.Withdraw{
			CoinTypeID:    info.CoinTypeID,
			CoinName:      coin.Name,
			CoinLogo:      coin.Logo,
			CoinUnit:      coin.Unit,
			Amount:        info.Amount,
			CreatedAt:     info.CreatedAt,
			Address:       acc.Address,
			AddressLabels: "TODO: to be filled",
			State:         state, // TODO: get transactions for Transferring/TransactionFail state
			Message:       messageMap[info.ID],
		})
	}

	return withdraws, total, nil
}

func GetIntervalWithdraws(
	ctx context.Context, appID, userID string, start, end uint32, offset, limit int32,
) (
	[]*npool.Withdraw, uint32, error,
) {
	return nil, 0, fmt.Errorf("NOT IMPLEMENTED")
}
