package forkliftcmd

import (
	"time"

	"github.com/ahl5esoft/golang-underscore"
	"github.com/ghodss/yaml"
	"github.com/n0rad/go-erlog/logs"
)

//var once sync.Once
//var commandConfig *ForkliftCommandConfig

type fcConfig ForkliftCommand

type forkliftCommandConfigSvc interface {
	SetDefaultCommand(commandName string, commandCwd string) ForkliftCommand
	findCommand(cmdName string, config interface{}) (configCmd ForkliftCommand)
	FindLocalCommand(cmdName string) (configCmd ForkliftCommand)
	FindRemoteCommand(cmdName string) (configCmd ForkliftCommand)
}

type ForkliftCommand struct {
	Shortname string        `json:"shortname",yaml:"shortname"`
	Path      string        `json:"path",yaml:"path"`
	Args      string        `json:"args,omitempty",yaml:"args,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty",yaml:"timeout,omitempty"`
	Cwd       string        `json:"cwd",yaml:"cwd"`
	Oneshot   bool          `json:"oneshot",yaml:"oneshot"`
}

type ForkliftCommandConfig struct {
	defaultCommand ForkliftCommand
	LocalConfig    []ForkliftCommand `json:"command,omitempty"`
	RemoteConfig   []ForkliftCommand `json:"remoteCommand,omitempty"`
}

func (fc *ForkliftCommand) UnmarshalJSON(b []byte) (err error) {
	fcConfigUnmarshal := fcConfig(*fc)
	if yaml.Unmarshal(b, &fcConfigUnmarshal) != nil {
		return err
	}
	fcConfigUnmarshal.Timeout = fcConfigUnmarshal.Timeout * time.Millisecond
	na := ForkliftCommand(fcConfigUnmarshal)
	*fc = na
	return nil
}

func MapConfigFile(fileContent []byte) (config ForkliftCommandConfig, err error) {
	if len(fileContent) < 1 {
		return config, nil
	}
	err = yaml.Unmarshal(fileContent, &config)
	logs.WithField("config", config).
		WithField("fileContent", string(fileContent)).
		Debug("Loading Command config")
	return config, err
}

func NewForkliftCommandConfig() ForkliftCommandConfig {
	commandConfig := ForkliftCommandConfig{}
	return commandConfig
}

func (cfg *ForkliftCommandConfig) SetDefaultCommand(commandName string, commandCwd string) ForkliftCommand {
	cfg.defaultCommand = ForkliftCommand{
		Shortname: "default",
		Path:      commandName,
		Timeout:   0,
		Oneshot:   true,
		Cwd:       commandCwd,
	}
	return cfg.defaultCommand
}

func (cfg *ForkliftCommandConfig) findCommand(cmdName string, config interface{}) (configCmd ForkliftCommand) {
	if cmdName == "" || config == nil {
		configCmd = cfg.defaultCommand
	} else {
		configCmd = config.(ForkliftCommand)
	}
	logs.WithField("config", configCmd).
		WithField("fullconfig", cfg).
		Debug("Finding config")
	return configCmd
}

func (cfg *ForkliftCommandConfig) FindLocalCommand(cmdName string) (configCmd ForkliftCommand) {
	tmpConfigRemoteCmd := underscore.FindBy(cfg.LocalConfig, map[string]interface{}{"shortname": cmdName})
	return cfg.findCommand(cmdName, tmpConfigRemoteCmd)
}

func (cfg *ForkliftCommandConfig) FindRemoteCommand(cmdName string) (configCmd ForkliftCommand) {
	tmpConfigRemoteCmd := underscore.FindBy(cfg.RemoteConfig, map[string]interface{}{"shortname": cmdName})
	return cfg.findCommand(cmdName, tmpConfigRemoteCmd)
}
