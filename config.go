package main

import (
	"encoding/json"
	"errors"
	"os"
)

type ConfigData struct {
	Token          string   `json:"token"`
	GroupId        string   `json:"group_id"`
	StartCommands  []string `json:"start_commands"`
	WelcomeMessage string   `json:"welcome_message"`
}

func ConfigNew(path string) error {
	cfg := &ConfigData{
		Token:          "TO BE FILLED",
		GroupId:        "TO BE FILLED",
		StartCommands:  []string{"start", "begin", "hello"},
		WelcomeMessage: "Welcome!\nSend message \"hello\" to get started.",
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err = json.NewEncoder(f).Encode(cfg); err != nil {
		return err
	}

	return nil
}

func ConfigRead(path string) (*ConfigData, error) {
	cfg := new(ConfigData)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err = ConfigNew(path); err == nil {
				return nil, errors.New("config file created, fill it")
			}
		}

		return nil, err
	}
	defer func() { _ = f.Close() }()

	if err = json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
