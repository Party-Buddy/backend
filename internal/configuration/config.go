package configuration

import (
	"github.com/spf13/viper"
	"log"
)

const (
	appEnvPrefix    = "PARTY_BUDDY"
	appEnvDbPrefix  = "PARTY_BUDDY_DB"
	appEnvImgPrefix = "PARTY_BUDDY_IMG"
)

// configureEnvs maps the config values to proper environment variables
func configureEnvs() {
	_ = viper.BindEnv("server.host", appEnvPrefix+"_HOST")
	_ = viper.BindEnv("server.port", appEnvPrefix+"_PORT")

	_ = viper.BindEnv("db.host", appEnvDbPrefix+"_HOST")
	_ = viper.BindEnv("db.port", appEnvDbPrefix+"_PORT")
	_ = viper.BindEnv("db.name", appEnvDbPrefix+"_NAME")
	_ = viper.BindEnv("db.driver", appEnvDbPrefix+"_DRIVER")
	_ = viper.BindEnv("db.user", appEnvDbPrefix+"_USER")
	_ = viper.BindEnv("db.password", appEnvDbPrefix+"_PASSWORD")

	_ = viper.BindEnv("img.path", appEnvImgPrefix+"_PATH")
}

// ConfigureApp try to get configuration from ./configs/conf.[ext] file.
// According to viper documentation [ext] may be:
//
// - json
//
// - yml
//
// - and some others
//
// Also here configureEnvs is called.
// If some envs is set then configuration will have the values specified in the appropriate envs
func ConfigureApp() {
	viper.SetConfigName("conf")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("config were not provided")
	}

	configureEnvs()
	viper.AutomaticEnv()
}
