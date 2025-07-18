package lib

import (
    "github.com/spf13/viper"
)

func LoadConfig(path string) error {
    viper.SetConfigFile(path)
    if err := viper.ReadInConfig(); err != nil {
        return err
    }
    return nil
}