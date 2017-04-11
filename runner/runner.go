package runner

import (
	"os/exec"
	"sort"
	"syscall"
	"time"

	"github.com/n0rad/go-erlog/logs"
	"github.com/nyodas/forklift/logstreamer"
)

type Runner struct {
	commandName string
	commandCwd  string
	Args        []string
	TermStdOut  logstreamer.LogStreamer
	TermStdErr  logstreamer.LogStreamer
	process     *exec.Cmd
	Timeout     time.Duration
	Status      int
	Oneshot     bool
	exitCode    []int
}

type RunnerSvc interface {
	SetLogger(stdOut logstreamer.LogStreamer, stdErr logstreamer.LogStreamer)
	Prepare()
	Start() int
	Stop()
	LaunchTimeout() *time.Timer
	ExecLoop()
}

func NewRunner(name string, commandCwd string, commandArgs []string) *Runner {
	runner := &Runner{
		commandName: name,
		commandCwd:  commandCwd,
		Args:        commandArgs,
		exitCode:    []int{120, 121, 122, 123, 124, 125, 126, 127},
		Oneshot:     false,
	}
	return runner
}

func (r *Runner) SetLogger(stdOut logstreamer.LogStreamer, stdErr logstreamer.LogStreamer) {
	r.TermStdOut = stdOut
	r.TermStdErr = stdErr
	r.process.Stdout = stdOut
	r.process.Stderr = stdErr
}

func (r *Runner) Prepare() {
	cmd := exec.Command(
		r.commandName,
		r.Args...,
	)
	stdOut := logstreamer.NewLogStreamerTerm("stdout", false, r.commandName)
	stdErr := logstreamer.NewLogStreamerTerm("stderr", false, r.commandName)
	r.process = cmd
	r.SetLogger(stdOut, stdErr)
	r.process.Dir = r.commandCwd
}

func (r *Runner) Start() int {
	r.Status = 0
	var timer *time.Timer
	logs.WithField("command", r.commandName).
		WithField("args", r.Args).
		WithField("timeous", r.Timeout).
		Debug("Executing command")

	if err := r.process.Start(); err != nil {
		logs.WithE(err).WithField("command", r.commandName).
			WithField("args", r.Args).
			Error("Error executing command")
	}
	if r.Timeout != 0 {
		timer = r.LaunchTimeout()
	}
	if err := r.process.Wait(); err != nil {
		logs.WithE(err).WithField("command", r.commandName).
			WithField("args", r.Args).
			Error("Error executing command")
		err_cmd := err.(*exec.ExitError)
		r.Status = err_cmd.Sys().(syscall.WaitStatus).ExitStatus()
	}
	if r.Timeout != 0 {
		timer.Stop()
	}
	logs.WithField("command", r.commandName).
		WithField("process", r.process.ProcessState).
		WithField("exitcode", r.Status).
		Debug("Command exited")
	r.TermStdOut.Flush()
	r.TermStdErr.Flush()
	return r.Status
}

func (r *Runner) Stop() {
	logs.WithField("command", r.commandName).Info("Stoping command")
	r.process.Process.Kill()
	r.TermStdOut.Flush()
	r.TermStdErr.Flush()
}

func (r *Runner) LaunchTimeout() *time.Timer {
	var timer *time.Timer
	logs.WithField("timeout", r.Timeout).Debug("Setting timeout")
	timer = time.AfterFunc(r.Timeout, func() {
		logs.WithField("timeout", r.Timeout).Debug("Timeout triggered")
		timer.Stop()
		r.Stop()
	})
	return timer
}

func (r *Runner) ExecLoop() {
	restartIntTime := 0
	restartIntStatus := 0
	lastStart := time.Now()
	for {
		r.Prepare()
		logs.WithField("command", r.commandName).
			WithField("args", r.Args).
			WithField("restart", restartIntTime).
			WithField("oneshot", r.Oneshot).
			WithField("lastStart", time.Since(lastStart)).
			Debug("Restart")
		if time.Since(lastStart) < 1000*time.Millisecond {
			restartIntTime += 1
		} else {
			restartIntTime = 1
		}
		if sort.SearchInts(r.exitCode, r.Status) > 0 {
			restartIntStatus += 1
		} else if restartIntStatus > 0 {
			restartIntStatus -= 1
		}
		lastStart = time.Now()
		_ = r.Start()
		if restartIntTime == 3 || restartIntStatus == 3 || r.Oneshot {
			logs.WithField("restartTime", restartIntTime).
				WithField("restartFail", restartIntStatus).
				WithField("exitCode", r.Status).
				WithField("lastStart", time.Since(lastStart)).
				Info("Restart Limit Reached")
			break
		}
	}
}
