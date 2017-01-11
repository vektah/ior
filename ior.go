package main

import (
	"bytes"
	"crypto/sha1"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sync/singleflight"
)

var ignoreDirs []string = []string{"vendor", "node_modules", "bundler", "public", "assets"}
var once = singleflight.Group{}

var bindir = flag.String("bindir", mustCwd()+"/bin", "The location to put the binary")
var binary = flag.String("binary", "", "The binary to run after installing")
var port = flag.String("port", "3000", "The port the app should listen on, will set PORT")
var upstream = flag.String("upstream", "", "Where to connect to access the app")
var listen = flag.String("listen", ":3030", "Where to listen")

func mustCwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

func main() {
	runtime.GOMAXPROCS(1)
	flag.Parse()

	if *binary == "" {
		*binary = *bindir + "/" + filepath.Base(mustCwd())
	}
	if *upstream == "" {
		*upstream = "http://localhost:" + *port
	}
	rpURL, err := url.Parse(*upstream)
	if err != nil {
		log.Fatal(err)
	}

	log.Print(http.ListenAndServe(*listen, reloadMiddleware(httputil.NewSingleHostReverseProxy(rpURL))))
}

func reloadMiddleware(next http.Handler) http.Handler {
	var lastHash []byte

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err, _ := once.Do("all", func() (interface{}, error) {
			hash := getHash()
			if !bytes.Equal(lastHash, hash) {
				println("Change detected, recompiling")
				err := install()
				if err != nil {
					return nil, err
				}

				println("Reloading")
				err = reload()
				if err != nil {
					return nil, err
				}
				lastHash = hash
			}

			return nil, nil
		})

		if err != nil {
			http.Error(w, err.Error(), 400)
		}

		next.ServeHTTP(w, r)
	})
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
