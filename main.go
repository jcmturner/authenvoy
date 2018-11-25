package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jcmturner/authenvoy/config"
)

const appTitle = "Authentication Envoy"

var buildhash = "Not set"
var buildtime = "Not set"
var version = "Not set"

func main() {
	version := flag.Bool("version", false, "Print version information.")
	logs := flag.String("log-dir", "./", "Directory to output logs to.")
	port := flag.Int("port", 8088, "Port to listen on loopback.")
	krbconf := flag.String("krb5-conf", "./krb5.conf", "Path to krb5.conf file.")
	//configPath := flag.String("config", "./awskmsluks-config.json", "Specify the path to the configuration file.")
	flag.Parse()

	// Print version information and exit.
	if *version {
		fmt.Fprintf(os.Stderr, versionStr())
		os.Exit(0)
	}

	c := config.New(*port, *krbconf, *logs)
}

// Version returns the version number, hash from git and the time of the build.
func versionInfo() (string, string, time.Time) {
	bt, _ := time.Parse(time.RFC3339, buildtime)
	return version, buildhash, bt
}

// VersionStr returns the version number, hash from git and the time of the build in a pretty formatted string.
func versionStr() string {
	v, bh, bt := versionInfo()
	return fmt.Sprintf("%s Version Information:\nVersion:\t%s\nBuild hash:\t%s\nBuild time:\t%v\n", appTitle, v, bh, bt)
}
