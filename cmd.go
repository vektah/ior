package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const escape = "\x1b"

var cmd *exec.Cmd
var currentCloser func()

func install() error {
	if _, err := os.Stat(*bindir); os.IsNotExist(err) {
		os.Mkdir(*bindir, 0755)
	}

	cmd := exec.Command("go", "install", "-v")
	cmd.Env = append(os.Environ(), "GOBIN="+*bindir, "PORT="+*port)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()

	b := &bytes.Buffer{}

	go func() {
		copyColor(os.Stdout, stdout)
	}()

	go func() {
		copyColor(os.Stderr, io.TeeReader(stderr, b))
	}()

	err = cmd.Run()
	if _, ok := err.(*exec.ExitError); ok && b.Len() > 0 {
		return errors.New(b.String())
	}
	return err
}

func copyColor(dst io.Writer, src io.Reader) error {
	b := make([]byte, 4*1024)
	var err error

	for {
		nr, er := src.Read(b)
		if nr > 0 {
			fmt.Fprint(dst, "\x1b[34m")
			nw, ew := dst.Write(b[0:nr])
			fmt.Fprint(dst, "\x1b[0m")
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return err
}

func stop() bool {
	stopped := false
	if cmd != nil {
		cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
		cmd = nil
		stopped = true
	}
	if currentCloser != nil {
		currentCloser()
		currentCloser = nil
	}

	return stopped
}

func reload() error {
	cmd = exec.Command(*binary, flag.Args()...)
	cmd.Env = append(os.Environ(), "PORT="+*port)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	currentCloser = func() {
		stdout.Close()
		stderr.Close()
	}

	go func() {
		io.Copy(os.Stdout, stdout)
	}()

	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	err = cmd.Start()
	if err != nil {
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

	return nil
}
