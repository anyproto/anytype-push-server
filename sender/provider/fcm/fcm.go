package fcm

import (
	"context"

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

	android, err := newSender(conf.CredentialsFile.Android)
	if err != nil {
		return err
	}
	s.RegisterProvider(domain.PlatformAndroid, android)

	ios, err := newSender(conf.CredentialsFile.IOS)
	if err != nil {
		return err
	}
	s.RegisterProvider(domain.PlatformIOS, ios)
	return
}

func (f *fcm) Name() (name string) {
	return CName
}

func newSender(credentialsFile string) (sender.Provider, error) {
	opt := option.WithCredentialsFile(credentialsFile)
	fcmApp, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, err
	}
	client, err := fcmApp.Messaging(context.Background())
	if err != nil {
		return nil, err
	}
	return &fcmSender{client: client}, nil
}

type fcmSender struct {
	client *messaging.Client
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
		var response *messaging.BatchResponse
		if response, err = f.client.SendEachForMulticast(ctx, buildFcmMessage(message)); err != nil {
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

func buildFcmMessage(message domain.Message) *messaging.MulticastMessage {
	return &messaging.MulticastMessage{
		Tokens: message.Tokens,
		Data:   message.Data,
	}
}
