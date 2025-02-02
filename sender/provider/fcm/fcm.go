package fcm

import (
	"context"

	"github.com/anyproto/any-sync/app"

	"github.com/anyproto/anytype-push-server/domain"
	"github.com/anyproto/anytype-push-server/sender"
)

const CName = "push.provider.fcm"

func New() FCM {
	return new(fcm)
}

type FCM interface {
	sender.Provider
	app.Component
}

type fcm struct {
}

func (f *fcm) Init(a *app.App) (err error) {
	s := a.MustComponent(sender.CName).(sender.Sender)
	s.RegisterProvider(domain.PlatformIOS, f)
	s.RegisterProvider(domain.PlatformAndroid, f)
	return
}

func (f *fcm) Name() (name string) {
	return CName
}

func (f *fcm) SendMessage(ctx context.Context, message domain.Message, onInvalid func(token string)) (err error) {
	//TODO implement me
	panic("implement me")
}
