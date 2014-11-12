package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pearkes/dnsimple"
)

var verbose = flag.Bool("v", false, "Use verbose output")
var list = flag.Bool("l", false, "List domains.")
var update = flag.String("u", "", "Update or create record. The format is 'domain name type oldvalue newvlaue ttl'. Use - for oldvalue to create a new record.")
var del = flag.String("d", "", "Delete record. The format is 'domain name type value'")

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
	if *update != "" {
		id, err := createOrUpdate(client, *update)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not get create or update:", err)
		} else {
			fmt.Printf("record written with id %s\n", id)
		}
		return
	}
	if *del != "" {
		id, err := deleteRecord(client, *del)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not delete:", err)
		} else {
			fmt.Printf("record deleted with id %s\n", id)
		}
		return
	}
	for _, domain := range flag.Args() {
		records, err := client.GetRecords(domain)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not get records:", err)
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

func createOrUpdate(client *dnsimple.Client, message string) (string, error) {
	pieces := strings.Split(message, " ")
	if len(pieces) != 6 {
		return "", fmt.Errorf("expected space seperated domain, name, type, oldvalue, newvalue, ttl")
	}
	domain := pieces[0]
	var changeRecord dnsimple.ChangeRecord
	changeRecord.Name = pieces[1]
	changeRecord.Type = pieces[2]
	changeRecord.Value = pieces[3]
	newRecord := changeRecord
	newRecord.Value = pieces[4]
	newRecord.Ttl = pieces[5]
	id, err := getRecordIdByValue(client, domain, &changeRecord)
	if err != nil {
		return "", err
	}
	var respId string
	if id == "" {
		respId, err = client.CreateRecord(domain, &newRecord)
	} else {
		respId, err = client.UpdateRecord(domain, id, &newRecord)
	}
	if err != nil {
		return "", err
	}
	return respId, nil
}

func deleteRecord(client *dnsimple.Client, message string) (string, error) {
	pieces := strings.Split(message, " ")
	if len(pieces) != 4 {
		return "", fmt.Errorf("expected space seperated domain, name, type, value")
	}
	domain := pieces[0]
	var changeRecord dnsimple.ChangeRecord
	changeRecord.Name = pieces[1]
	changeRecord.Type = pieces[2]
	changeRecord.Value = pieces[3]
	id, err := getRecordIdByValue(client, domain, &changeRecord)
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("could not find record")
	}
	err = client.DestroyRecord(domain, id)
	return id, err
}

func getRecordIdByValue(client *dnsimple.Client, domain string, changeRecord *dnsimple.ChangeRecord) (string, error) {
	records, err := client.GetRecords(domain)
	if err != nil {
		return "", err
	}
	var id string
	for _, record := range records {
		if record.Name == changeRecord.Name && record.RecordType == changeRecord.Type && record.Content == changeRecord.Value {
			id = record.StringId()
			break
		}
	}
	return id, nil
}
