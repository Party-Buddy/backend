package configuration

import (
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

// GetDBConfig provides the *pgxpool.Config by
// creating a connection string and using pgxpool.ParseConfig.
// Connection string is created with info from viper
func GetDBConfig() (*pgxpool.Config, error) {
	user := viper.GetString("db.user")
	if user == "" {
		return nil, ErrDBUserInfoNotProvided
	}
	host := viper.GetString("db.host")
	if host == "" {
		return nil, ErrDBHostNotProvided
	}
	port := viper.GetString("db.port")
	if port == "" {
		return nil, ErrDBPortNotProvided
	}
	dbname := viper.GetString("db.name")
	if dbname == "" {
		return nil, ErrDBNameNotProvided
	}
	pass := viper.GetString("db.password")
	if pass == "" {
		return nil, ErrDBUserInfoNotProvided
	}
	var connectionString = fmt.Sprintf("postgres://%v:%v@%v:%v/%v", user, pass, host, port, dbname)

	conf, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
