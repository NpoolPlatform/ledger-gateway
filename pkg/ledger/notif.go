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

func CreateNotif(
	ctx context.Context,
	appID, userID, langID, userName string,
	eventType notifmgrpb.EventType,
) {
	templateInfo, err := thirdcli.GetNotifTemplateOnly(ctx, &thirdmgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		LangID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: langID,
		},
		UsedFor: &commonpb.Uint32Val{
			Op:    cruder.EQ,
			Value: uint32(eventType.Number()),
		},
	})
	if err != nil {
		logger.Sugar().Errorw("sendNotif", "error", err.Error())
		return
	}
	if templateInfo == nil {
		logger.Sugar().Errorw("sendNotif", "error", "template not exist")
		return
	}

	content := thirdpkg.ReplaceVariable(templateInfo.Content, &userName, nil)
	useTemplate := true

	_, err = notifcli.CreateNotif(ctx, &notifmgrpb.NotifReq{
		AppID:       &appID,
		UserID:      &userID,
		LangID:      &langID,
		EventType:   &eventType,
		UseTemplate: &useTemplate,
		Title:       &templateInfo.Title,
		Content:     &content,
		Channels:    []channelpb.NotifChannel{channelpb.NotifChannel_ChannelEmail},
	})
	if err != nil {
		logger.Sugar().Errorw("sendNotif", "error", err.Error())
		return
	}
}
