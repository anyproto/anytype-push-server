package fcm

type configSource interface {
	GetFCM() Config
}

type Config struct {
	CredentialsFile struct {
		IOS     string `yaml:"ios"`
		Android string `yaml:"android"`
	} `yaml:"credentialsFile"`
}
