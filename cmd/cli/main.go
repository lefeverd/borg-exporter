package main

import "github.com/lefeverd/borg-exporter/internal/web"

// Version will hold the version of the application, set at build time
var Version = "dev"

func main() {
	web.Execute(Version)
}
