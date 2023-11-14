package internal

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"party-buddy/internal/configuration"
	"party-buddy/internal/handlers"
)

func Main() {
	configuration.ConfigureApp()
	handler := handlers.ConfigureMux()

	host := viper.GetString("server.host")
	if host == "" {
		viper.SetDefault("server.host", "localhost")
	}
	port := viper.GetString("server.port")
	if port == "" {
		viper.SetDefault("server.port", "8081")
	}

	log.Printf("Listening on port %s", port)
	log.Printf("Open http://%s:%s in the browser", host, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", host, port), handler))
}
