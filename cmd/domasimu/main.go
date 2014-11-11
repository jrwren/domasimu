package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pearkes/dnsimple"
)

var verbose = flag.Bool("v", false, "verbose")
var list = flag.Bool("l", false, "list domains")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, ".domasimurc config file example:")
		toml.NewEncoder(os.Stderr).Encode(Config{"you@example.com", "TOKENHERE1234"})
	}
	flag.Parse()
	user, token, err := getCreds()
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not read config", err)
		return
	}
	client, err := dnsimple.NewClient(user, token)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not connect to dnsimple", err)
		return
	}
	if *list {
		domains, err := client.GetDomains()
		if err != nil {
			fmt.Fprintln(os.Stderr, "could get domains %v", err)
			return
		}
		for _, domain := range domains {
			if *verbose {
				fmt.Println(domain.Name, domain.ExpiresOn)
			} else {
				fmt.Println(domain.Name)
			}
		}
		return
	}
	for _, domain := range flag.Args() {
		records, err := client.GetRecords(domain)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could get records %v", err)
			continue
		}
		for _, record := range records {
			if *verbose {
				fmt.Println(record.Name, record.RecordType, record.Content, record.Ttl, record.Prio)
			} else {
				fmt.Println(record.Name, record.RecordType, record.Content)
			}
		}
	}
}

func getCreds() (string, string, error) {
	configFileName := os.Getenv("DOMASIMU_CONF")
	if configFileName == "" {
		configFileName = filepath.Join(os.Getenv("HOME"), ".domasimurc")
	}
	var config Config
	_, err := toml.DecodeFile(configFileName, &config)
	if err != nil {
		return "", "", err
	}
	return config.User, config.Token, nil
}

type Config struct {
	User  string
	Token string
}
