package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fatih/color"
)

func main() {
	commands := os.Args[1:]
	wg := sync.WaitGroup{}

	for i, command := range commands {
		cr := NewCommandRunner(i, command)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cr.RunCommand(); err != nil {
				fmt.Println(err)
			}
		}()
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-exit
		fmt.Println("\nReceived term signal")
	}()

	wg.Wait()
}

type commandRunner struct {
	id      int
	command string
}

func NewCommandRunner(id int, command string) *commandRunner {
	return &commandRunner{id: id, command: command}
}

var bgColors = []color.Attribute{
	color.BgHiGreen,
	color.BgHiRed,
	color.BgHiYellow,
	color.BgHiBlue,
	color.BgHiMagenta,
	color.BgHiCyan,
}

// var fgColors = []color.Attribute{
// 	color.FgRed,
// 	color.FgGreen,
// 	color.FgYellow,
// 	color.FgBlue,
// 	color.FgMagenta,
// 	color.FgCyan,
// }

func (cr *commandRunner) RunCommand() error {
	indexColor := color.New(bgColors[cr.id%len(bgColors)], color.FgHiBlack)
	sprintIndex := func() string {
		return indexColor.Sprintf("[%d]", cr.id)
	}

	fmt.Fprintln(os.Stderr, sprintIndex(), color.HiBlackString("$ %s", cr.command))
	cmd := exec.Command("sh", "-c", cr.command)

	_, _ = cmd.StdinPipe()
	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	for _, reader := range []io.ReadCloser{stderr, stdout} {
		wg.Add(1)
		go func(reader io.ReadCloser) {
			defer wg.Done()
			scanner := bufio.NewScanner(reader)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				m := scanner.Text()
				if reader == stderr {
					fmt.Fprintln(os.Stderr, sprintIndex(), m)
				}
				if reader == stdout {
					fmt.Fprintln(os.Stdout, sprintIndex(), m)
				}
			}
		}(reader)
	}
	wg.Wait()
	cmd.Wait()
	fmt.Fprintf(os.Stderr, `%s Command "%s" (pid: %d) exited with exit status: %s.`+"\n", sprintIndex(), cr.command, cmd.ProcessState.Pid(), cmd.ProcessState)
	return nil
}
