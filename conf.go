package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pelletier/go-toml"
)

type config struct {
	Token      string
	Prefix     string
	InviteUrl  string `comment:"URL for a support server invite"`
	EmoteGuild string
}

func getConfig(path string) error {
	tomlConf, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		newConf, err := toml.Marshal(conf)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(path, newConf, 0666)
		if err != nil {
			return err
		}
		return errors.New(
			fmt.Sprintf("no config found. please review the new values at %s and try again", path))
	}
	if err != nil {
		return err
	}
	err = toml.Unmarshal(tomlConf, &conf)
	if err != nil {
		return err
	}
	return nil
}
