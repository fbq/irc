package bot

import (
	"encoding/json"
	"os"
)

type BotConfig struct {
	Server string
	Nick string
	Pass string
	User string
	Info string
	Port uint16
	Channels []string
}

func ConfigBotFromFile(path string) (*BotConfig, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	configuration := BotConfig{}
	err = decoder.Decode(&configuration)

	if err != nil {
		return nil, err
	}

	return &configuration, nil

}

