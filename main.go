package main

import (
	"github.com/spf13/viper"
	"log"
	"party-buddy/internal"
)

const (
	AppEnvPrefix   = "PARTY_BUDDY"
	AppEnvDbPrefix = "PARTY_BUDDY_DB"
)

func configureEnvs() {
	_ = viper.BindEnv("server.host", AppEnvPrefix+"_HOST")
	_ = viper.BindEnv("server.port", AppEnvPrefix+"_PORT")
	_ = viper.BindEnv("db.host", AppEnvDbPrefix+"_HOST")
	_ = viper.BindEnv("db.port", AppEnvDbPrefix+"_PORT")
	_ = viper.BindEnv("db.name", AppEnvDbPrefix+"_NAME")
	_ = viper.BindEnv("db.driver", AppEnvDbPrefix+"_DRIVER")
	_ = viper.BindEnv("db.user", AppEnvDbPrefix+"_USER")
	_ = viper.BindEnv("db.password", AppEnvDbPrefix+"_PASSWORD")
}

func main() {
	viper.SetConfigName("conf")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("config were not provided")
	}

	configureEnvs()
	viper.AutomaticEnv()

	internal.Main()
}
