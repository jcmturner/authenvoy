package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jcmturner/authenvoy/config"
	"github.com/jcmturner/authenvoy/httphandling"
)

const appTitle = "Authentication Envoy"

var buildhash = "Not set"
var buildtime = "Not set"

func main() {
	version := flag.Bool("version", false, "Print version information.")
	logs := flag.String("log-dir", "./", "Directory to output logs to.")
	port := flag.Int("port", 8088, "Port to listen on loopback.")
	krbconf := flag.String("krb5-conf", "./krb5.conf", "Path to krb5.conf file.")
	flag.Parse()

	// Print version information and exit.
	if *version {
		fmt.Fprintf(os.Stderr, versionStr())
		os.Exit(0)
	}

	c, err := config.New(*port, *krbconf, *logs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s configuration error: %v", appTitle, err)
		os.Exit(1)
	}

	socket := fmt.Sprintf("%s:%d", "127.0.0.1", c.Port)
	c.ApplicationLogf(versionStr())
	err = http.ListenAndServe(socket, httphandling.NewRouter(c))
	log.Fatalf("%s exit: %v\n", appTitle, err)
}

// Version returns the version number, hash from git and the time of the build.
func versionInfo() (string, time.Time) {
	bt, _ := time.Parse(time.RFC3339, buildtime)
	return buildhash, bt
}

// VersionStr returns the version number, hash from git and the time of the build in a pretty formatted string.
func versionStr() string {
	bh, bt := versionInfo()
	return fmt.Sprintf("%s Version Information:\nBuild hash:\t%s\nBuild time:\t%v\n", appTitle, bh, bt)
}
