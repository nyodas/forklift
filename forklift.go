package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/mgutz/str"
	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
	"github.com/nyodas/forklift/forkliftcmd"
	forkliftHttp "github.com/nyodas/forklift/http"
	forkliftRunner "github.com/nyodas/forklift/runner"
)

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")
var commandName = flag.String("c", "/bin/ls", "Command to run")
var commandCwd = flag.String("cwd", "/", "Cwd for the command")
var commandArgs = flag.String("cargs", "", "Args for the default background command")
var logLevel = flag.String("L", "info", "Loglevel  (default is INFO)")
var execProc = flag.Bool("e", false, "Exec background process")
var configPath = flag.String("config", "", "Config file path")
var postStopHook = flag.String("S", "", "PostStopHook when exec.")

func main() {
	flag.Parse()
	level, err := logs.ParseLevel(*logLevel)
	if err != nil {
		logs.WithField("value", logLevel).Fatal("Unknown log level")
	}
	logs.SetLevel(level)
	file, err := loadConfig(*configPath)
	if err != nil {
		logs.WithE(err).WithField("configfile", configPath).
			Error("Config file empty or missing")
	}

	cmdConfig, err := forkliftcmd.MapConfigFile(file)
	if err != nil {
		logs.WithE(err).WithField("configfile", configPath).
			WithField("config", cmdConfig).
			Fatal("Failed to map forkliftcmd config file")
	}
	logs.WithE(err).WithField("configfile", configPath).
		WithField("config", cmdConfig).Debug("cmdConfig Content")
	defaultCmd := cmdConfig.SetDefaultCommand(*commandName, *commandCwd)

	if *execProc {
		if file != nil && *commandArgs == "" {
			runBackgroundCmds(cmdConfig.LocalConfig)
		} else {
			defaultCmd.Args = *commandArgs
			defaultCmd.PostStopHook = *postStopHook
			runBackgroundCmd(defaultCmd)
		}
	}

	forkliftHttpHandler := forkliftHttp.Handler{
		ForkliftConfig: &cmdConfig,
	}
	http.HandleFunc("/echo", forkliftHttpHandler.ExecRemoteCmd)
	http.HandleFunc("/exec", forkliftHttpHandler.ExecRemoteCmd)
	http.HandleFunc("/healthz", forkliftHttpHandler.Healthz)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func runBackgroundCmds(cmdConfigs []forkliftcmd.ForkliftCommand) {
	for _, v := range cmdConfigs {
		runBackgroundCmd(v)
	}
}

func runBackgroundCmd(cmdConfig forkliftcmd.ForkliftCommand) {
	runner := forkliftRunner.NewRunner(cmdConfig.Path, cmdConfig.Cwd, str.ToArgv(cmdConfig.Args))
	runner.Timeout = cmdConfig.Timeout
	runner.Oneshot = cmdConfig.Oneshot
	runner.PostStopHook = cmdConfig.PostStopHook
	done := make(chan struct{})
	go func() {
		runner.ExecLoop()
		close(done)
	}()
	go exitHandler(done, runner)
}

func exitHandler(done chan struct{}, runner *forkliftRunner.Runner) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	for {
		select {
		case <-interrupt:
			runner.Stop()
			os.Exit(1)
		case <-done:
			logs.WithField("exitcode", runner.Status).
				Debug("We're done.Exiting")
			os.Exit(runner.Status)
			return
		}
	}
}

func loadConfig(configPath string) (file []byte, err error) {
	if configPath == "" {
		return nil, errors.New("No config file defined")
	}
	logs.WithField("configfile", configPath).Debug("Loading config")
	file, err = ioutil.ReadFile(configPath)
	return file, err
}
