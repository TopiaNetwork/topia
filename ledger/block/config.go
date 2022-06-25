package block

import (
	"github.com/spf13/viper"
	"log"
)

func main() {
	//设置要读取配置文件的名字
	viper.SetConfigName("config")
	//配置文件格式 yaml 也可以支持json等格式,按需设置
	viper.SetConfigType("yaml")
	//viper.SetConfigType("yaml")
	//分割符 默认用. 这样子配置文件的完整名字为 config.yaml
	viper.AddConfigPath(".")
	//也可以设置默认值
	viper.SetDefault("version.index_version", 10)
	//开始载入配置文件
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("read config failed: %v", err)
	}
	//查看配置文件

}