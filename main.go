package main

import (
	"os"
	"strconv"

	"flag"
	"fmt"
	"github.com/qb0C80aE/clay/db"
	"github.com/qb0C80aE/clay/server"
)

func main() {

	version := flag.Bool("v", false, "prints current Clay version")
	flag.Parse()
	if *version {
		fmt.Printf("%s\n", versionString)
		os.Exit(0)
	}

	host := "localhost"
	port := "8080"

	if h := os.Getenv("HOST"); h != "" {
		host = h
	}

	if p := os.Getenv("PORT"); p != "" {
		if _, err := strconv.Atoi(p); err == nil {
			port = p
		}
	}

	os.Setenv("HOST", host)
	os.Setenv("PORT", port)

	database := db.Connect()
	s := server.Setup(database)

	s.Run(fmt.Sprintf("%s:%s", host, port))

}
