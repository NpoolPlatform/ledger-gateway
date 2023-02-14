//nolint:dupl
package ledger

import (
	"context"

	notifmgrpb "github.com/NpoolPlatform/message/npool/notif/mgr/v1/notif"
	notifcli "github.com/NpoolPlatform/notif-middleware/pkg/client/notif"
	thirdpkg "github.com/NpoolPlatform/third-middleware/pkg/template"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	commonpb "github.com/NpoolPlatform/message/npool"

	channelpb "github.com/NpoolPlatform/message/npool/notif/mgr/v1/channel"

	"github.com/NpoolPlatform/message/npool/third/mgr/v1/usedfor"

	notifchannelpb "github.com/NpoolPlatform/message/npool/notif/mgr/v1/notif/notifchannel"
	notifchannelcli "github.com/NpoolPlatform/notif-middleware/pkg/client/notif/notifchannel"

	frontendmgrpb "github.com/NpoolPlatform/message/npool/third/mgr/v1/template/frontend"
	frontendcli "github.com/NpoolPlatform/third-middleware/pkg/client/template/frontend"

	emailmgrpb "github.com/NpoolPlatform/message/npool/third/mgr/v1/template/email"
	emailcli "github.com/NpoolPlatform/third-middleware/pkg/client/template/email"

	smsmgrpb "github.com/NpoolPlatform/message/npool/third/mgr/v1/template/sms"
	smscli "github.com/NpoolPlatform/third-middleware/pkg/client/template/sms"

	g11ncli "github.com/NpoolPlatform/g11n-middleware/pkg/client/applang"
	g11npb "github.com/NpoolPlatform/message/npool/g11n/mgr/v1/applang"
)

const LIMIT = uint32(1000)

func CreateNotif(
	ctx context.Context,
	appID, userID string,
	userName, amount, coinUnit *string,
	eventType usedfor.UsedFor,
) {
	channelInfos, _, err := notifchannelcli.GetNotifChannels(ctx, &notifchannelpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		EventType: &commonpb.Uint32Val{
			Op:    cruder.EQ,
			Value: uint32(eventType),
		},
	}, 0, int32(len(channelpb.NotifChannel_value)))
	if err != nil {
		logger.Sugar().Errorw("CreateNotif", "error", err.Error())
		return
	}
	notifReq := []*notifmgrpb.NotifReq{}

	for _, val := range channelInfos {
		if val.Channel == channelpb.NotifChannel_ChannelFrontend {
			notifReq = append(
				notifReq,
				createFrontendNotif(ctx, appID, userID, userName, amount, coinUnit, eventType)...,
			)
		}
		if val.Channel == channelpb.NotifChannel_ChannelEmail {
			email := createEmailNotif(ctx, appID, userID, userName, amount, coinUnit, eventType)
			if email != nil {
				notifReq = append(notifReq, email)
			}
		}
		if val.Channel == channelpb.NotifChannel_ChannelSMS {
			sms := createSMSNotif(ctx, appID, userID, userName, amount, coinUnit, eventType)
			if sms != nil {
				notifReq = append(notifReq, sms)
			}
		}
	}

	_, err = notifcli.CreateNotifs(ctx, notifReq)
	if err != nil {
		logger.Sugar().Errorw("CreateNotif", "error", err.Error())
		return
	}
}
func createFrontendNotif(
	ctx context.Context,
	appID, userID string,
	userName, amount, coinUnit *string,
	eventType usedfor.UsedFor,
) []*notifmgrpb.NotifReq {
	offset := uint32(0)
	limit := LIMIT
	notifChannel := channelpb.NotifChannel_ChannelFrontend
	notifReq := []*notifmgrpb.NotifReq{}
	for {
		templateInfos, _, err := frontendcli.GetFrontendTemplates(ctx, &frontendmgrpb.Conds{
			AppID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: appID,
			},
			UsedFor: &commonpb.Uint32Val{
				Op:    cruder.EQ,
				Value: uint32(eventType.Number()),
			},
		}, offset, limit)
		offset += limit
		if err != nil {
			logger.Sugar().Errorw("CreateNotif", "error", err.Error())
			return nil
		}
		if len(templateInfos) == 0 {
			break
		}
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
				Channel:     &notifChannel,
			})
		}
	}
	return notifReq
}

func createEmailNotif(
	ctx context.Context,
	appID, userID string,
	userName, amount, coinUnit *string,
	eventType usedfor.UsedFor,
) *notifmgrpb.NotifReq {
	notifChannel := channelpb.NotifChannel_ChannelEmail

	mainLang, err := g11ncli.GetLangOnly(ctx, &g11npb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		Main: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: true,
		},
	})
	if err != nil {
		logger.Sugar().Errorw("sendNotif", "error", err)
		return nil
	}
	if mainLang == nil {
		logger.Sugar().Errorw(
			"sendNotif",
			"AppID", appID,
			"error", "MainLang is invalid")
		return nil
	}
	templateInfo, err := emailcli.GetEmailTemplateOnly(ctx, &emailmgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UsedFor: &commonpb.Int32Val{
			Op:    cruder.EQ,
			Value: int32(eventType.Number()),
		},
		LangID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: mainLang.GetLangID(),
		},
	})
	if err != nil {
		logger.Sugar().Errorw("CreateNotif", "error", err.Error())
		return nil
	}

	if templateInfo == nil {
		logger.Sugar().Errorw(
			"CreateNotif",
			"AppID",
			appID,
			"UsedFor",
			eventType.String(),
			"LangID",
			mainLang.LangID,
			"error",
			"template not exists",
		)
		return nil
	}

	useTemplate := true
	content := thirdpkg.ReplaceVariable(
		templateInfo.Body,
		userName,
		nil,
		amount,
		coinUnit,
		nil,
		nil,
		nil,
	)

	return &notifmgrpb.NotifReq{
		AppID:       &appID,
		UserID:      &userID,
		LangID:      &templateInfo.LangID,
		EventType:   &eventType,
		UseTemplate: &useTemplate,
		Title:       &templateInfo.Subject,
		Content:     &content,
		Channel:     &notifChannel,
	}
}

func createSMSNotif(
	ctx context.Context,
	appID, userID string,
	userName, amount, coinUnit *string,
	eventType usedfor.UsedFor,
) *notifmgrpb.NotifReq {
	notifChannel := channelpb.NotifChannel_ChannelSMS
	mainLang, err := g11ncli.GetLangOnly(ctx, &g11npb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		Main: &commonpb.BoolVal{
			Op:    cruder.EQ,
			Value: true,
		},
	})
	if err != nil {
		logger.Sugar().Errorw("sendNotif", "error", err)
		return nil
	}
	if mainLang == nil {
		logger.Sugar().Errorw(
			"sendNotif",
			"AppID", appID,
			"error", "MainLang is invalid")
		return nil
	}
	templateInfo, err := smscli.GetSMSTemplateOnly(ctx, &smsmgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UsedFor: &commonpb.Int32Val{
			Op:    cruder.EQ,
			Value: int32(eventType.Number()),
		},
		LangID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: mainLang.GetLangID(),
		},
	})
	if err != nil {
		logger.Sugar().Errorw("CreateNotif", "error", err.Error())
		return nil
	}

	if templateInfo == nil {
		logger.Sugar().Errorw(
			"CreateNotif",
			"AppID",
			appID,
			"UsedFor",
			eventType.String(),
			"LangID",
			mainLang.LangID,
			"error",
			"template not exists",
		)
		return nil
	}

	useTemplate := true
	content := thirdpkg.ReplaceVariable(
		templateInfo.Message,
		userName,
		nil,
		amount,
		coinUnit,
		nil,
		nil,
		nil,
	)

	return &notifmgrpb.NotifReq{
		AppID:       &appID,
		UserID:      &userID,
		LangID:      &templateInfo.LangID,
		EventType:   &eventType,
		UseTemplate: &useTemplate,
		Title:       &templateInfo.Subject,
		Content:     &content,
		Channel:     &notifChannel,
	}
}
