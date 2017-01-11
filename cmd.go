package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"net"
	"os"
	"os/exec"
	"time"
)

var currentCmd *exec.Cmd
var currentCloser func()

func install() error {
	if _, err := os.Stat(*bindir); os.IsNotExist(err) {
		os.Mkdir(*bindir, 0755)
	}

	cmd, buf, closer, err := cmd("go", "install", "-v")
	if err != nil {
		return err
	}
	defer closer()

	err = cmd.Run()
	if _, ok := err.(*exec.ExitError); ok && buf.Len() > 0 {
		return errors.New(buf.String())
	}
	return err
}

func cmd(name string, args ...string) (*exec.Cmd, *bytes.Buffer, func(), error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "GOBIN="+*bindir, "PORT="+*port)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	errbuf := &bytes.Buffer{}

	go func() {
		io.Copy(os.Stdout, stdout)
	}()

	go func() {
		io.Copy(errbuf, io.TeeReader(stderr, os.Stderr))
	}()

	return cmd, errbuf, func() {
		stdout.Close()
		stderr.Close()
	}, nil

}

func reload() error {
	if currentCmd != nil {
		currentCmd.Process.Kill()
		_ = currentCmd.Wait()
		currentCloser()
	}

	cmd, buf, closer, err := cmd(*binary, flag.Args()...)
	if err != nil {
		return err
	}

	err = cmd.Start()
	if _, ok := err.(*exec.ExitError); ok && buf.Len() > 0 {
		return errors.New(buf.String())
	} else if err != nil {
		return err
	}

	for {
		conn, err := net.Dial("tcp", "127.0.0.1:"+*port)
		if err == nil {
			conn.Close()
			break
		}

		time.Sleep(15 * time.Millisecond)
	}

	currentCmd = cmd
	currentCloser = closer

	return nil
}
