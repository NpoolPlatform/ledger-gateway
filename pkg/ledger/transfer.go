package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/NpoolPlatform/message/npool/third/mgr/v1/usedfor"

	thirdmwcli "github.com/NpoolPlatform/third-middleware/pkg/client/verify"

	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	"github.com/NpoolPlatform/message/npool"
	"github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	constant "github.com/NpoolPlatform/ledger-gateway/pkg/message/const"

	signmethodpb "github.com/NpoolPlatform/message/npool/appuser/mgr/v2/signmethod"

	appusermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"

	appusermgrcli "github.com/NpoolPlatform/appuser-manager/pkg/client/kyc"
	appusermgrpb "github.com/NpoolPlatform/message/npool/appuser/mgr/v2/kyc"

	accountmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/transfer"
	accountmgrpb "github.com/NpoolPlatform/message/npool/account/mgr/v1/transfer"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/v2"

	ledgermgrpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"

	"go.opentelemetry.io/otel"
	scodes "go.opentelemetry.io/otel/codes"
)

//nolint:funlen
func CreateTransfer(
	ctx context.Context,
	appID,
	userID,
	account string,
	accountType signmethodpb.SignMethodType,
	verificationCode,
	targetUserID,
	amount,
	coinTypeID string,
) (*ledger.Transfer, error) {
	var err error

	_, span := otel.Tracer(constant.ServiceName).Start(ctx, "CreateTransfer")
	defer span.End()

	defer func() {
		if err != nil {
			span.SetStatus(scodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	user, err := appusermwcli.GetUser(ctx, appID, userID)
	if err != nil {
		return nil, err
	}
	if accountType == signmethodpb.SignMethodType_Google {
		account = user.GetGoogleSecret()
	}

	if err := thirdmwcli.VerifyCode(
		ctx,
		appID,
		account,
		verificationCode,
		accountType,
		usedfor.UsedFor_Transfer,
	); err != nil {
		return nil, err
	}

	kyc, err := appusermgrcli.GetKycOnly(ctx, &appusermgrpb.Conds{
		AppID: &npool.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &npool.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
	})
	if err != nil {
		return nil, err
	}
	if kyc == nil {
		return nil, fmt.Errorf("kyc not added")
	}

	if kyc.State != appusermgrpb.KycState_Approved {
		return nil, fmt.Errorf("kyc state is not approved")
	}

	exist, err := accountmwcli.ExistTransferConds(ctx, &accountmgrpb.Conds{
		AppID: &npool.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &npool.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
		TargetUserID: &npool.StringVal{
			Op:    cruder.EQ,
			Value: targetUserID,
		},
	})
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, fmt.Errorf("target user not set")
	}

	targetUser, err := appusermwcli.GetUser(ctx, appID, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil {
		return nil, fmt.Errorf("target user not found")
	}

	coin, err := coininfocli.GetCoinInfo(ctx, coinTypeID)
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid coin")
	}

	out := ledgermgrpb.IOType_Outcoming
	outIoExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","TargetUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		appID,
		userID,
		targetUserID,
		coin.Name,
		amount,
		time.Now(),
	)

	subType := ledgermgrpb.IOSubType_Transfer

	in := ledgermgrpb.IOType_Incoming
	inIoExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","FromUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		appID,
		targetUserID,
		userID,
		coin.Name,
		amount,
		time.Now(),
	)

	createdAt := uint32(time.Now().Unix())

	err = ledgermwcli.BookKeeping(ctx, []*ledgermgrpb.DetailReq{
		{
			AppID:      &appID,
			UserID:     &userID,
			CoinTypeID: &coinTypeID,
			IOType:     &out,
			IOSubType:  &subType,
			Amount:     &amount,
			IOExtra:    &outIoExtra,
			CreatedAt:  &createdAt,
		}, {
			AppID:      &appID,
			UserID:     &targetUserID,
			CoinTypeID: &coinTypeID,
			IOType:     &in,
			IOSubType:  &subType,
			Amount:     &amount,
			IOExtra:    &inIoExtra,
			CreatedAt:  &createdAt,
		},
	})
	if err != nil {
		return nil, err
	}

	return &ledger.Transfer{
		CoinTypeID:         coin.ID,
		CoinName:           coin.Name,
		CoinLogo:           coin.Logo,
		CoinUnit:           coin.Unit,
		Amount:             amount,
		CreatedAt:          createdAt,
		TargetUserID:       targetUserID,
		TargetEmailAddress: targetUser.EmailAddress,
		TargetPhoneNO:      targetUser.PhoneNO,
		TargetUsername:     targetUser.Username,
		TargetFirstName:    targetUser.FirstName,
		TargetLastName:     targetUser.LastName,
	}, nil
}
