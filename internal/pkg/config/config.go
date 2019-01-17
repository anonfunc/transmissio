package config

import (
	"log"

	"github.com/spf13/viper"
)

func Config() {
	setDefaults()
	viper.SetConfigName("config")
	viper.AddConfigPath("/config")
	viper.AddConfigPath("$HOME/.transmissio")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if err := viper.WriteConfigAs("./config.yaml"); err != nil {
			log.Fatalf("Could not write config file to config.yaml: %s\n", err)
			return
		}
		log.Fatalf("Wrote config file to config.yaml.  Please set values.\n")
	}

	viper.SetEnvPrefix("t")
	viper.AutomaticEnv()
}

func setDefaults() {
	viper.SetDefault("blackhole", "/blackhole")
	viper.SetDefault("downloadTo", "/download")
	viper.SetDefault("host", "")
	viper.SetDefault("port", "9091")
	viper.SetDefault("oauth_token", "Get from https://app.put.io/settings/account/oauth/apps")
}
