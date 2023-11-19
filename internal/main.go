package internal

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"party-buddy/internal/api/handlers"
	"party-buddy/internal/configuration"
	"party-buddy/internal/db"
	"party-buddy/internal/validate"
)

// isImagePathAccessible tries to create a file by provided image path
func isImagePathAccessible() error {
	imgDir := configuration.GetImgDirectory()
	if _, err := os.Stat(imgDir); os.IsNotExist(err) {
		if err := os.MkdirAll(imgDir, 0700); err != nil {
			return err
		}
	}

	testFilePath := imgDir + string(os.PathSeparator) + "test.png"

	file, err := os.OpenFile(testFilePath, os.O_RDWR|os.O_CREATE, 0700)
	if err != nil {
		return err
	}

	_, err = file.WriteString("hello world")
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}

	err = os.Remove(testFilePath)
	return err
}

func Main() {
	configuration.ConfigureApp()

	err := isImagePathAccessible()
	if err != nil {
		log.Fatalf("Failed to test image path accessibility: %v", err.Error())
	}

	dbPoolConf, err := db.GetDBConfig()
	if err != nil {
		log.Fatalf("Failed to init db config: %v", err.Error())
	}

	dbpool, err := db.InitDBPool(context.Background(), dbPoolConf)
	if err != nil {
		log.Fatalf("Failed to init db pool: %v", err.Error())
	}

	ctx := context.Background()

	factory := validate.NewValidationFactory()
	ctx = validate.NewContext(ctx, factory)

	handler := handlers.ConfigureMux(ctx, &dbpool)

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
