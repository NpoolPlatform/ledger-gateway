package ledger

import (
	"context"
	"fmt"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermgrwithdrawcli "github.com/NpoolPlatform/ledger-manager/pkg/client/withdraw"
	ledgermgrwithdrawpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/withdraw"

	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	billingcli "github.com/NpoolPlatform/cloud-hashing-billing/pkg/client"
	billingpb "github.com/NpoolPlatform/message/npool/cloud-hashing-billing"

	reviewmgrpb "github.com/NpoolPlatform/message/npool/review/mgr/v2"
	reviewcli "github.com/NpoolPlatform/review-service/pkg/client"
	reviewconst "github.com/NpoolPlatform/review-service/pkg/const"

	constant "github.com/NpoolPlatform/ledger-gateway/pkg/message/const"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"
)

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

	stateMap := map[string]npool.WithdrawState{}
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

	for _, r := range reviews {
		if waitTsMap[r.ObjectID] < rejectedTsMap[r.ObjectID] {
			stateMap[r.ObjectID] = npool.WithdrawState_Reviewing
			continue
		}
		stateMap[r.ObjectID] = npool.WithdrawState_Rejected
	}

	for _, r := range reviews {
		switch r.State {
		case reviewconst.StateApproved:
			fallthrough //nolint
		case reviewmgrpb.ReviewState_Approved.String():
			stateMap[r.ObjectID] = npool.WithdrawState_Successful
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
			State:         state, // TODO: get transactions for Transfering/TransactionFail state
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
