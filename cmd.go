package main

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/singleflight"
)

type Daemon struct {
	cmd           *exec.Cmd
	currentCloser func()
	lastHash      []byte
	once          singleflight.Group
}

func (d *Daemon) Refresh() error {
	_, err, _ := d.once.Do("reload", func() (interface{}, error) {
		hash := getHash()
		if !bytes.Equal(d.lastHash, hash) {
			start := time.Now()
			if d.stop() {
				fmt.Printf("\x1b[36mStopped in %s.\n\x1b[0m", time.Since(start).String())
			}
			start = time.Now()
			err := d.install()
			if err != nil {
				return nil, err
			}

			fmt.Printf("\x1b[36mRebuilt in %s.\n\x1b[0m", time.Since(start).String())
			err = d.reload()
			if err != nil {
				return nil, err
			}
			d.lastHash = hash
		} else if !d.running() {
			d.reload()
		}

		return nil, nil
	})

	return err
}

func (d *Daemon) install() error {
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

func (d *Daemon) stop() bool {
	stopped := false
	if d.cmd != nil {
		d.cmd.Process.Signal(syscall.SIGTERM)
		_ = d.cmd.Wait()
		d.cmd = nil
		stopped = true
	}
	if d.currentCloser != nil {
		d.currentCloser()
		d.currentCloser = nil
	}

	return stopped
}

func (d *Daemon) running() bool {
	return d.cmd != nil
}

func (d *Daemon) reload() error {
	d.cmd = exec.Command(*binary, flag.Args()...)
	d.cmd.Env = append(os.Environ(), "PORT="+*port)
	stdout, err := d.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := d.cmd.StderrPipe()
	if err != nil {
		return err
	}
	d.currentCloser = func() {
		stdout.Close()
		stderr.Close()
	}

	go func() {
		io.Copy(os.Stdout, stdout)
	}()

	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	err = d.cmd.Start()
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

func getHash() []byte {
	s := sha1.New()
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if info.Name() != "." && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			for _, ignoreDir := range ignoreDirs {
				if info.Name() == ignoreDir {
					return filepath.SkipDir
				}
			}
		}

		if strings.HasSuffix(info.Name(), ".go") {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(s, f)
			if err != nil {
				return err
			}
		}

		return nil
	})
	return s.Sum([]byte{})
}
