package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
)

type Config struct {
	AppId           string `json:",omitempty"`
	AppSecret       string `json:",omitempty"`
	PageId          string
	ShortLivedToken string
	LongLivedToken  string `json:",omitempty"`
}

var FB Config
var file string

func Init(configFile string) error {
	if configFile == "" {
		return errors.New("Config file is required")
	}
	file = configFile
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return errors.Wrap(err, "Read config file")
	}

	err = json.Unmarshal(content, &FB)
	if err != nil {
		return errors.Wrap(err, "Unmarshal config file")
	}

	if FB.ShortLivedToken == "" {
		return errors.New("ShortLivedToken is required")
	}
	if FB.PageId == "" {
		return errors.New("PageId is required")
	}

	return nil
}

func Save() error {
	content, err := json.MarshalIndent(FB, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Marshal config file")
	}
	err = ioutil.WriteFile(file, content, 0660)
	if err != nil {
		return errors.Wrap(err, "Write config file")
	}
	return nil
}
