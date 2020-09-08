package main

import (
	"encoding/json"
	"errors"
	"os"
)

const configFilePath string = "config.json"

type Config struct {
	Token   string `json:"token"`
	Version string `json:"version"`
	GroupId string `json:"group_id"`
}

func readConfig(path string) (*Config, error) {
	cfg := new(Config)

	f, err := os.Open(path)
	if err != nil {
		// Файл не создан. Создаем
		if os.IsNotExist(err) {
			f, err = os.Create(path)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			if err = json.NewEncoder(f).Encode(cfg); err != nil {
				return nil, err
			}

			return nil, errors.New("new config was created, fill it")
		}

		// Файл создан. Какая-то другая ошибка
		return nil, err
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
