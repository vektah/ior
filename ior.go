package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
)

var ignoreDirs []string = []string{"vendor", "node_modules", "bundler", "public", "assets"}

var bindir = flag.String("bindir", mustCwd()+"/bin", "The location to put the binary")
var binary = flag.String("binary", "", "The binary to run after installing")
var port = flag.String("port", "3000", "The port the app should listen on, will set PORT")
var upstream = flag.String("upstream", "", "Where to connect to access the app")
var listen = flag.String("listen", ":3030", "Where to listen")
var race = flag.Bool("race", false, "Build binary with -race instrumentation")

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

	d := &Daemon{
		race: *race,
	}
	d.Refresh()

	log.Print(http.ListenAndServe(*listen, reloadMiddleware(d, NewForwarder(rpURL))))
}

func reloadMiddleware(d *Daemon, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := d.Refresh()

		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		next.ServeHTTP(w, r)
	})
}
