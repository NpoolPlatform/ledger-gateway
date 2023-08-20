package transfer

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	usercodemwcli "github.com/NpoolPlatform/basal-middleware/pkg/client/usercode"
	usercodemwpb "github.com/NpoolPlatform/message/npool/basal/mw/v1/usercode"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	appusermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"

	kycmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/kyc"
	kycmwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/kyc"

	accountmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/transfer"
	accountmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/transfer"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"
	statementmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/statement"

	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	statementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/statement"

	ledgerpb "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

//nolint:funlen,gocyclo
func (h *Handler) CreateTransfer(ctx context.Context) (*npool.Transfer, error) {
	user, err := appusermwcli.GetUser(ctx, *h.AppID, *h.UserID)
	if err != nil {
		return nil, err
	}

	if h.AccountType == basetypes.SignMethod_Google {
		h.Account = &user.GoogleSecret
	}

	if err := usercodemwcli.VerifyUserCode(ctx, &usercodemwpb.VerifyUserCodeRequest{
		Prefix:      basetypes.Prefix_PrefixUserCode.String(),
		AppID:       *h.AppID,
		Account:     *h.Account,
		AccountType: h.AccountType,
		UsedFor:     basetypes.UsedFor_Transfer,
		Code:        *h.VerificationCode,
	}); err != nil {
		return nil, err
	}

	kyc, err := kycmwcli.GetKycOnly(ctx, &kycmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
	})
	if err != nil {
		return nil, err
	}
	if kyc == nil {
		return nil, fmt.Errorf("kyc not added")
	}

	if kyc.State != basetypes.KycState_Approved {
		return nil, fmt.Errorf("kyc state is not approved")
	}

	ledger, err := ledgermwcli.GetLedgerOnly(ctx, &ledgermwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		UserID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.UserID,
		},
		CoinTypeID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.CoinTypeID,
		},
	})
	if err != nil {
		return nil, err
	}
	if ledger == nil {
		return nil, fmt.Errorf("ledger not exist")
	}

	ad, err := decimal.NewFromString(*h.Amount)
	if err != nil {
		return nil, err
	}

	spendable, err := decimal.NewFromString(ledger.Spendable)
	if err != nil {
		return nil, err
	}
	if spendable.Cmp(ad) < 0 {
		return nil, fmt.Errorf("insufficient funds")
	}

	exist, err := accountmwcli.ExistTransferConds(ctx, &accountmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		UserID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.UserID,
		},
		TargetUserID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.TargetUserID,
		},
	})
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, fmt.Errorf("target user not set")
	}

	targetUser, err := appusermwcli.GetUser(ctx, *h.AppID, *h.TargetUserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil {
		return nil, fmt.Errorf("target user not found")
	}

	coin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		CoinTypeID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.CoinTypeID,
		},
	})
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid coin")
	}

	now := uint32(time.Now().Unix())

	out := ledgerpb.IOType_Outcoming
	outIoExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","TargetUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		*h.AppID,
		*h.UserID,
		*h.TargetUserID,
		coin.Name,
		*h.Amount,
		now,
	)

	subType := ledgerpb.IOSubType_Transfer

	in := ledgerpb.IOType_Incoming
	inIoExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","FromUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		*h.AppID,
		*h.TargetUserID,
		*h.UserID,
		coin.Name,
		h.Amount,
		now,
	)

	_, err = statementmwcli.CreateStatements(ctx, []*statementpb.StatementReq{
		{
			AppID:      h.AppID,
			UserID:     h.UserID,
			CoinTypeID: h.CoinTypeID,
			IOType:     &out,
			IOSubType:  &subType,
			Amount:     h.Amount,
			IOExtra:    &outIoExtra,
			CreatedAt:  &now,
		}, {
			AppID:      h.AppID,
			UserID:     h.TargetUserID,
			CoinTypeID: h.CoinTypeID,
			IOType:     &in,
			IOSubType:  &subType,
			Amount:     h.Amount,
			IOExtra:    &inIoExtra,
			CreatedAt:  &now,
		},
	})
	if err != nil {
		return nil, err
	}

	return &npool.Transfer{
		CoinTypeID:         coin.CoinTypeID,
		CoinName:           coin.Name,
		DisplayNames:       coin.DisplayNames,
		CoinLogo:           coin.Logo,
		CoinUnit:           coin.Unit,
		Amount:             *h.Amount,
		CreatedAt:          now,
		TargetUserID:       *h.TargetUserID,
		TargetEmailAddress: targetUser.EmailAddress,
		TargetPhoneNO:      targetUser.PhoneNO,
		TargetUsername:     targetUser.Username,
		TargetFirstName:    targetUser.FirstName,
		TargetLastName:     targetUser.LastName,
	}, nil
}
