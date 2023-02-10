package ledger

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"
	channelpb "github.com/NpoolPlatform/message/npool/notif/mgr/v1/channel"
	notifmgrpb "github.com/NpoolPlatform/message/npool/notif/mgr/v1/notif"
	thirdmgrpb "github.com/NpoolPlatform/message/npool/third/mgr/v1/template/notif"
	notifcli "github.com/NpoolPlatform/notif-middleware/pkg/client/notif"
	thirdcli "github.com/NpoolPlatform/third-middleware/pkg/client/template/notif"
	thirdpkg "github.com/NpoolPlatform/third-middleware/pkg/template/notif"
)

const LIMIT = uint32(1000)

func CreateNotif(
	ctx context.Context,
	appID, userID string,
	userName, amount, coinUnit *string,
	eventType notifmgrpb.EventType,
) {
	offset := uint32(0)
	limit := LIMIT
	for {
		templateInfos, _, err := thirdcli.GetNotifTemplates(ctx, &thirdmgrpb.Conds{
			AppID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: appID,
			},
			UsedFor: &commonpb.Uint32Val{
				Op:    cruder.EQ,
				Value: uint32(eventType.Number()),
			},
		}, offset, limit)
		if err != nil {
			logger.Sugar().Errorw("CreateNotif", "error", err.Error())
			return
		}
		if len(templateInfos) == 0 {
			logger.Sugar().Errorw("CreateNotif", "error", "template not exist")
			return
		}

		notifReq := []*notifmgrpb.NotifReq{}
		useTemplate := true

		for _, val := range templateInfos {
			content := thirdpkg.ReplaceVariable(
				val.Content,
				userName,
				nil,
				amount,
				coinUnit,
				nil,
				nil,
				nil,
			)

			notifReq = append(notifReq, &notifmgrpb.NotifReq{
				AppID:       &appID,
				UserID:      &userID,
				LangID:      &val.LangID,
				EventType:   &eventType,
				UseTemplate: &useTemplate,
				Title:       &val.Title,
				Content:     &content,
				Channels:    []channelpb.NotifChannel{channelpb.NotifChannel_ChannelEmail},
			})
		}

		_, err = notifcli.CreateNotifs(ctx, notifReq)
		if err != nil {
			logger.Sugar().Errorw("CreateNotif", "error", err.Error())
			return
		}
	}
}
