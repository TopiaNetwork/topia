package block

import (
	"github.com/spf13/viper"
	"log"
)

func main() {

	viper.SetConfigName("config")

	viper.SetConfigType("yaml")
	//viper.SetConfigType("yaml")

	viper.AddConfigPath(".")

	viper.SetDefault("version.index_version", 10)

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("read config failed: %v", err)
	}


}