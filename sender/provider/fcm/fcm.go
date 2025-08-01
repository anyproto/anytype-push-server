package fcm

import (
	"context"
	"fmt"
	"maps"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"go.uber.org/zap"
	"google.golang.org/api/option"

	"github.com/anyproto/anytype-push-server/domain"
	"github.com/anyproto/anytype-push-server/sender"
)

const CName = "push.provider.fcm"

var log = logger.NewNamed(CName)

func New() FCM {
	return new(fcm)
}

type FCM interface {
	app.Component
}

type fcm struct {
}

func (f *fcm) Init(a *app.App) (err error) {
	s := a.MustComponent(sender.CName).(sender.Sender)
	conf := a.MustComponent("config").(configSource).GetFCM()

	android, err := newSender(conf, domain.PlatformAndroid, conf.CredentialsFile.Android)
	if err != nil {
		return err
	}
	s.RegisterProvider(domain.PlatformAndroid, android)

	ios, err := newSender(conf, domain.PlatformIOS, conf.CredentialsFile.IOS)
	if err != nil {
		return err
	}
	s.RegisterProvider(domain.PlatformIOS, ios)
	return
}

func (f *fcm) Name() (name string) {
	return CName
}

func newSender(config Config, platform domain.Platform, credentialsFile string) (sender.Provider, error) {
	opt := option.WithCredentialsFile(credentialsFile)
	fcmApp, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, err
	}
	client, err := fcmApp.Messaging(context.Background())
	if err != nil {
		return nil, err
	}
	return &fcmSender{client: client, config: config, platform: platform}, nil
}

type fcmSender struct {
	client   *messaging.Client
	platform domain.Platform
	config   Config
}

const batchSize = 500

func (f *fcmSender) SendMessage(ctx context.Context, message domain.Message, onInvalid func(token string)) (err error) {
	nextBatch := message.Tokens
	for len(nextBatch) > 0 {
		if len(nextBatch) > batchSize {
			message.Tokens = nextBatch[:batchSize]
			nextBatch = nextBatch[batchSize:]
		} else {
			message.Tokens = nextBatch
			nextBatch = nil
		}

		var multicastMessage *messaging.MulticastMessage
		switch f.platform {
		case domain.PlatformIOS:
			if message.Silent {
				multicastMessage = f.buildFcmIosSilentMessage(message)
			} else {
				multicastMessage = f.buildFcmIosMessage(message)
			}
		case domain.PlatformAndroid:
			if message.Silent {
				multicastMessage = f.buildFcmAndroidSilentMessage(message)
			} else {
				multicastMessage = f.buildFcmAndroidMessage(message)
			}
		default:
			return fmt.Errorf("unexpected platform %v", f.platform)
		}

		var response *messaging.BatchResponse
		if response, err = f.client.SendEachForMulticast(ctx, multicastMessage); err != nil {
			return err
		}
		for i, resp := range response.Responses {
			if resp.Error == nil {
				continue
			}
			log.Warn("fcm resp error", zap.Error(resp.Error))
			if messaging.IsInvalidArgument(resp.Error) || messaging.IsUnregistered(resp.Error) {
				onInvalid(message.Tokens[i])
				log.Info("mark token as invalid", zap.String("token", message.Tokens[i]))
			} else {
				log.Warn("fcm returned error", zap.Error(resp.Error), zap.String("token", message.Tokens[i]))
			}
		}
		log.Info("push sent", zap.Int("success", response.SuccessCount), zap.Int("failure", response.FailureCount))
	}
	return nil
}

func (f *fcmSender) buildFcmIosMessage(message domain.Message) *messaging.MulticastMessage {
	return &messaging.MulticastMessage{
		Tokens: message.Tokens,
		Data:   message.Data,
		Notification: &messaging.Notification{
			Title:    f.config.DefaultMessage.Title,
			Body:     f.config.DefaultMessage.Body,
			ImageURL: f.config.DefaultMessage.ImageUrl,
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					MutableContent: true,
				},
			},
		},
	}
}

func (f *fcmSender) buildFcmIosSilentMessage(message domain.Message) *messaging.MulticastMessage {
	return &messaging.MulticastMessage{
		Tokens: message.Tokens,
		Data:   message.Data,
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					MutableContent: true,
				},
			},
		},
	}
}

func (f *fcmSender) buildFcmAndroidMessage(message domain.Message) *messaging.MulticastMessage {
	var data = make(map[string]string)
	maps.Copy(data, message.Data)
	data["x-any-title"] = f.config.DefaultMessage.Title
	data["x-any-body"] = f.config.DefaultMessage.Body
	data["x-any-image-url"] = f.config.DefaultMessage.ImageUrl
	return &messaging.MulticastMessage{
		Tokens: message.Tokens,
		Data:   data,
	}
}

func (f *fcmSender) buildFcmAndroidSilentMessage(message domain.Message) *messaging.MulticastMessage {
	return &messaging.MulticastMessage{
		Tokens: message.Tokens,
		Data:   message.Data,
	}
}
