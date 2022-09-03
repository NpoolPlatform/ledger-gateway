package ledger

import (
	"context"
	"fmt"
	"time"

	thirdgwcli "github.com/NpoolPlatform/third-gateway/pkg/client"
	thirdgwconst "github.com/NpoolPlatform/third-gateway/pkg/const"

	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	"github.com/NpoolPlatform/message/npool"
	"github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	constant "github.com/NpoolPlatform/ledger-gateway/pkg/message/const"

	signmethodpb "github.com/NpoolPlatform/message/npool/appuser/mgr/v2/signmethod"

	"go.opentelemetry.io/otel"

	scodes "go.opentelemetry.io/otel/codes"

	appusermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"

	appusermgrcli "github.com/NpoolPlatform/appuser-manager/pkg/client/kyc"
	appusermgrpb "github.com/NpoolPlatform/message/npool/appuser/mgr/v2/kyc"

	accountmgrcli "github.com/NpoolPlatform/account-manager/pkg/client/transfer"
	accountmgrpb "github.com/NpoolPlatform/message/npool/account/mgr/v1/transfer"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/v2"

	ledgermgrpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"

	appusergw "github.com/NpoolPlatform/appuser-gateway/pkg/ga"
)

//nolint:funlen,gocyclo
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

	switch accountType {
	case signmethodpb.SignMethodType_Mobile, signmethodpb.SignMethodType_Email:
		if err := thirdgwcli.VerifyCode(
			ctx,
			appID, userID,
			accountType, account, verificationCode,
			thirdgwconst.UsedForSetTransferTargetUser,
		); err != nil {
			return nil, err
		}
	case signmethodpb.SignMethodType_Google:
		_, err = appusergw.VerifyGoogleAuth(ctx, appID, userID, verificationCode)
		if err != nil {
			return nil, err
		}
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

	exist, err := accountmgrcli.ExistTransferConds(ctx, &accountmgrpb.Conds{
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
