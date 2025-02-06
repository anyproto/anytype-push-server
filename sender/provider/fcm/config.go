package fcm

type configSource interface {
	GetFCM() Config
}

type Config struct {
	CredentialsFile string `yaml:"credentialsFile"`
}
