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
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/transfer"

	kycmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/kyc"
	appusermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	kycmwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/kyc"
	appusermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"

	accountmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/transfer"
	accountmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/transfer"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"
	statementmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"

	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	statementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/NpoolPlatform/go-service-framework/pkg/pubsub"
	ledgerpb "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	eventmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/event"
)

type createHandler struct {
	*Handler
	user       *appusermwpb.User
	targetUser *appusermwpb.User
	appcoin    *appcoinmwpb.Coin
	info       *npool.Transfer
}

func (h *createHandler) rewardInternalTransfer() {
	if err := pubsub.WithPublisher(func(publisher *pubsub.Publisher) error {
		req := &eventmwpb.CalcluateEventRewardsRequest{
			AppID:       *h.AppID,
			UserID:      *h.UserID,
			EventType:   basetypes.UsedFor_InternalTransfer,
			Consecutive: 1,
		}
		return publisher.Update(
			basetypes.MsgID_CalculateEventRewardReq.String(),
			nil,
			nil,
			nil,
			req,
		)
	}); err != nil {
		logger.Sugar().Errorw(
			"InternalTransfer",
			"AppID", *h.AppID,
			"UserID", h.UserID,
			"Error", err,
		)
	}
}

func (h *createHandler) checkUser(ctx context.Context) error {
	user, err := appusermwcli.GetUser(ctx, *h.AppID, *h.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalid user")
	}
	h.user = user

	targetUser, err := appusermwcli.GetUser(ctx, *h.AppID, *h.TargetUserID)
	if err != nil {
		return err
	}
	if targetUser == nil {
		return fmt.Errorf("target user not found")
	}
	h.targetUser = targetUser

	switch *h.AccountType {
	case basetypes.SignMethod_Email:
		h.Account = &user.EmailAddress
	case basetypes.SignMethod_Mobile:
		h.Account = &user.PhoneNO
	case basetypes.SignMethod_Google:
		h.Account = &user.GoogleSecret
	default:
		return fmt.Errorf("invalid account type %v", *h.AccountType)
	}
	return nil
}

func (h *createHandler) verifyUserCode(ctx context.Context) error {
	if *h.AccountType == basetypes.SignMethod_Google {
		h.Account = &h.user.GoogleSecret
	}
	return usercodemwcli.VerifyUserCode(ctx, &usercodemwpb.VerifyUserCodeRequest{
		Prefix:      basetypes.Prefix_PrefixUserCode.String(),
		AppID:       *h.AppID,
		Account:     *h.Account,
		AccountType: *h.AccountType,
		UsedFor:     basetypes.UsedFor_Transfer,
		Code:        *h.VerificationCode,
	})
}

func (h *createHandler) checkKyc(ctx context.Context) error {
	kyc, err := kycmwcli.GetKycOnly(ctx, &kycmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
	})
	if err != nil {
		return err
	}
	if kyc == nil {
		return fmt.Errorf("kyc not added")
	}

	if kyc.State != basetypes.KycState_Approved {
		return fmt.Errorf("kyc state is not approved")
	}
	return nil
}

func (h *createHandler) checkTransferAmount(ctx context.Context) error {
	info, err := ledgermwcli.GetLedgerOnly(ctx, &ledgermwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
	})
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("ledger not exist")
	}

	spendable, err := decimal.NewFromString(info.Spendable)
	if err != nil {
		return err
	}
	if spendable.Cmp(*h.Amount) < 0 {
		return fmt.Errorf("insufficient funds")
	}
	return nil
}

func (h *createHandler) checkAccount(ctx context.Context) error {
	exist, err := accountmwcli.ExistTransferConds(ctx, &accountmwpb.Conds{
		AppID:        &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		TargetUserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.TargetUserID},
	})
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("target user not set")
	}
	return nil
}

func (h *createHandler) getCoin(ctx context.Context) error {
	coin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
	})
	if err != nil {
		return err
	}
	if coin == nil {
		return fmt.Errorf("invalid coin")
	}
	h.appcoin = coin
	return nil
}

func (h *Handler) CreateTransfer(ctx context.Context) (*npool.Transfer, error) {
	handler := &createHandler{
		Handler:    h,
		appcoin:    &appcoinmwpb.Coin{},
		user:       &appusermwpb.User{},
		targetUser: &appusermwpb.User{},
		info:       &npool.Transfer{},
	}

	if err := handler.checkUser(ctx); err != nil {
		return nil, err
	}
	if err := handler.verifyUserCode(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkKyc(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkTransferAmount(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkAccount(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCoin(ctx); err != nil {
		return nil, err
	}

	now := uint32(time.Now().Unix())
	subType := ledgerpb.IOSubType_Transfer
	amount := h.Amount.String()

	out := ledgerpb.IOType_Outcoming
	outIOExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","TargetUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		*h.AppID,
		*h.UserID,
		*h.TargetUserID,
		handler.appcoin.Name,
		*h.Amount,
		now,
	)
	in := ledgerpb.IOType_Incoming
	inIOExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","FromUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		*h.AppID,
		*h.TargetUserID,
		*h.UserID,
		handler.appcoin.Name,
		h.Amount,
		now,
	)

	_, err := statementmwcli.CreateStatements(ctx, []*statementpb.StatementReq{
		{
			AppID:      h.AppID,
			UserID:     h.UserID,
			CoinTypeID: h.CoinTypeID,
			IOType:     &out,
			IOSubType:  &subType,
			Amount:     &amount,
			IOExtra:    &outIOExtra,
			CreatedAt:  &now,
		},
		{
			AppID:      h.AppID,
			UserID:     h.TargetUserID,
			CoinTypeID: h.CoinTypeID,
			IOType:     &in,
			IOSubType:  &subType,
			Amount:     &amount,
			IOExtra:    &inIOExtra,
			CreatedAt:  &now,
		},
	})
	if err != nil {
		return nil, err
	}

	handler.rewardInternalTransfer()

	return &npool.Transfer{
		CoinTypeID:         handler.appcoin.CoinTypeID,
		CoinName:           handler.appcoin.Name,
		DisplayNames:       handler.appcoin.DisplayNames,
		CoinLogo:           handler.appcoin.Logo,
		CoinUnit:           handler.appcoin.Unit,
		Amount:             amount,
		CreatedAt:          now,
		TargetUserID:       *h.TargetUserID,
		TargetEmailAddress: handler.targetUser.EmailAddress,
		TargetPhoneNO:      handler.targetUser.PhoneNO,
		TargetUsername:     handler.targetUser.Username,
		TargetFirstName:    handler.targetUser.FirstName,
		TargetLastName:     handler.targetUser.LastName,
	}, nil
}
