package configuration

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
)

const (
	appEnvPrefix      = "PARTY_BUDDY"
	appEnvDbPrefix    = "PARTY_BUDDY_DB"
	appEnvImgPrefix   = "PARTY_BUDDY_IMG"
	appEnvOuterPrefix = "PARTY_BUDDY_OUTER"
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

	_ = viper.BindEnv("outer.host", appEnvOuterPrefix+"_HOST")
	_ = viper.BindEnv("outer.port", appEnvOuterPrefix+"_PORT")
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

var imgPath string

// GetImgDirectory returns the image directory path, which ends with os.PathSeparator
func GetImgDirectory() string {
	if imgPath != "" {
		return imgPath
	}

	imgPath = viper.GetString("img.path")
	if imgPath == "" {
		return ""
	}
	if !strings.HasSuffix(imgPath, string(os.PathSeparator)) {
		imgPath += string(os.PathSeparator)
	}
	return imgPath
}

func GenImgURI(imgID uuid.UUID) string {
	host := viper.GetString("outer.host")
	if host == "" {
		host = viper.GetString("server.host")
	}
	port := viper.GetString("outer.port")
	if port == "" {
		port = viper.GetString("server.port")
	}

	return fmt.Sprintf("http://%v:%v/api/v1/images/%v", host, port, imgID.String())
}
