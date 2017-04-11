package forkliftcmd

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

func TestMapConfigEmptyFileContent(t *testing.T) {
	defaultConfig := ForkliftCommandConfig{}
	config, _ := MapConfigFile([]byte(""))
	if !reflect.DeepEqual(defaultConfig, config) {
		t.Error("If file is empty it should return a standard ForkliftCommandConfig")
	}
}

func TestMapConfigEmptyFileContentErr(t *testing.T) {
	_, err := MapConfigFile([]byte(""))
	if err != nil {
		t.Error("Empty file content shouldn't throw an error")
	}
}

func TestMapConfigFileContent(t *testing.T) {
	configExpected := ForkliftCommand{
		Shortname: "test",
		Path:      "/bin/test",
		Timeout:   1050 * time.Millisecond,
		Cwd:       "/test",
	}
	config, err := MapConfigFile([]byte("remoteCommand:\n- shortname: test\n  timeout: 1050\n  path: /bin/test\n  cwd: /test"))
	assert.NoError(t, err, "Correctly formed file shouldn't throw an error")
	if assert.NotNil(t, config) {
		assert.Equal(t, config.RemoteConfig[0], configExpected)
	}
}

func TestMapConfigFileErroneousContent(t *testing.T) {
	_, err := MapConfigFile([]byte("remoteCommand:- shortname:test"))
	assert.Error(t, err, "Badly formed file should throw an error")

}

func TestNewForkliftCommandConfig(t *testing.T) {
	defaultConfig := ForkliftCommandConfig{}
	config := NewForkliftCommandConfig()
	if !reflect.DeepEqual(defaultConfig, config) {
		t.Error("NewForkliftCommandConfig should return a ForkliftCommandConfig")
	}
}

func TestSetDefaultCommand(t *testing.T) {
	config := NewForkliftCommandConfig()
	defaultCmd := config.SetDefaultCommand("defaultName", "defaultCwd")
	defaultForkliftCmd := ForkliftCommand{
		Shortname: "default",
		Path:      "defaultName",
		Timeout:   0,
		Oneshot:   true,
		Cwd:       "defaultCwd",
	}
	if !reflect.DeepEqual(defaultCmd, defaultForkliftCmd) {
		t.Error("SetDefaultCommand should return a proper ForkliftCommand")
	}
}

func TestFindLocalCommandConfig(t *testing.T) {
	config := NewForkliftCommandConfig()
	defaultCmd := config.SetDefaultCommand("defaultName", "defaultCwd")
	config.LocalConfig = []ForkliftCommand{
		{
			Shortname: "ls",
			Path:      "/bin/ls",
			Timeout:   0,
			Oneshot:   true,
			Cwd:       "/",
		},
		{
			Shortname: "sleep",
			Path:      "/bin/sleep",
			Timeout:   0,
			Oneshot:   true,
			Cwd:       "/",
		},
	}
	var matchTests = []struct {
		cmdName string
		out     ForkliftCommand
	}{
		{config.LocalConfig[0].Shortname, config.LocalConfig[0]},
		{config.LocalConfig[1].Shortname, config.LocalConfig[1]},
		{"", defaultCmd},
		{"no_match", defaultCmd},
	}
	for _, tt := range matchTests {
		foundCmd := config.FindLocalCommand(tt.cmdName)
		t.Logf("Testing %q", tt.cmdName)
		if !reflect.DeepEqual(tt.out, foundCmd) {
			t.Errorf("FindLocalCommand(%q) should return %v", tt.cmdName, tt.out)
		}
	}

}

func TestFindRemoteCommandConfig(t *testing.T) {
	config := NewForkliftCommandConfig()
	defaultCmd := config.SetDefaultCommand("defaultName", "defaultCwd")
	config.RemoteConfig = []ForkliftCommand{
		{
			Shortname: "ls",
			Path:      "/bin/ls",
			Timeout:   0,
			Oneshot:   true,
			Cwd:       "/",
		},
		{
			Shortname: "sleep",
			Path:      "/bin/sleep",
			Timeout:   100,
			Oneshot:   true,
			Cwd:       "/",
		},
	}
	var matchTests = []struct {
		cmdName string
		out     ForkliftCommand
	}{
		{config.RemoteConfig[0].Shortname, config.RemoteConfig[0]},
		{config.RemoteConfig[1].Shortname, config.RemoteConfig[1]},
		{"", defaultCmd},
		{"no_match", defaultCmd},
	}
	for _, tt := range matchTests {
		foundCmd := config.FindRemoteCommand(tt.cmdName)
		t.Logf("Testing %q", tt.cmdName)
		if !reflect.DeepEqual(tt.out, foundCmd) {
			t.Errorf("FindRemoteCommand(%q) should return %v", tt.cmdName, tt.out)
		}
	}

}

func TestFindCommandMatchConfig(t *testing.T) {
	config := NewForkliftCommandConfig()
	config.LocalConfig = []ForkliftCommand{
		{
			Shortname: "ls",
			Path:      "/bin/ls",
			Timeout:   0,
			Oneshot:   true,
			Cwd:       "/",
		},
		{
			Shortname: "sleep",
			Path:      "/bin/sleep",
			Timeout:   0,
			Oneshot:   true,
			Cwd:       "/",
		},
	}
	defaultCmd := config.SetDefaultCommand("defaultName", "defaultCwd")
	var matchTests = []struct {
		cmdName string
		input   interface{}
		out     ForkliftCommand
	}{
		{config.LocalConfig[0].Shortname, config.LocalConfig[0], config.LocalConfig[0]},
		{config.LocalConfig[1].Shortname, config.LocalConfig[1], config.LocalConfig[1]},
		{"", config.LocalConfig[1], defaultCmd},
		{config.LocalConfig[1].Shortname, nil, defaultCmd},
	}
	for _, tt := range matchTests {
		foundCmd := config.findCommand(tt.cmdName, tt.input)
		if !reflect.DeepEqual(tt.out, foundCmd) {
			t.Errorf("findCommand = %q should return %v", tt.cmdName, tt.out)
		}
	}
}
