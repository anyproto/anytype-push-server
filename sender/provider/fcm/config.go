package fcm

type configSource interface {
	GetFCM() Config
}

type Config struct {
	CredentialsFile struct {
		IOS     string `yaml:"ios"`
		Android string `yaml:"android"`
	} `yaml:"credentialsFile"`
	DefaultMessage struct {
		Title    string `yaml:"title"`
		Body     string `yaml:"body"`
		ImageUrl string `yaml:"imageUrl"`
	} `yaml:"defaultMessage"`
}
