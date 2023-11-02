package configuration

import (
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"log"
	"party-buddy/internal/handlers"
)

const (
	appEnvPrefix   = "PARTY_BUDDY"
	appEnvDbPrefix = "PARTY_BUDDY_DB"
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

// ConfigureMux configures the handlers for HTTP routes and methods
func ConfigureMux() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", handlers.IndexHandler).Methods("GET")
	return r
}
