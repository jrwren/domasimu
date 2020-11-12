// Copyright © 2014-2020 Jay R. Wren <jrwren@xmtp.net>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/dnsimple/dnsimple-go/dnsimple"
	"golang.org/x/oauth2"
)

var verbose = flag.Bool("v", false, "Use verbose output")
var list = flag.Bool("l", false, "List domains.")
var update = flag.String("u", "", "Update or create record. The format is 'domain name type oldvalue newvlaue ttl'.\n(use - for oldvalue to create a new record)")
var value = flag.String("value", "", "Alt value to use for create or update")
var del = flag.String("d", "", "Delete record. The format is 'domain name type value'")
var format = flag.String("f", "plain", "Output zones in {plain, json, table} format")
var typeR = flag.String("t", "", "record type to query for")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Println("domasimu <domainname> will list records in the domain")
		fmt.Println("")
		fmt.Println("domasimu -value='v=spf1 mx -all' -u 'example.com mail TXT - - 300' will add an SPF record")
		fmt.Println("")
		fmt.Fprintln(os.Stderr, "domasimu config file example:")
		err := toml.NewEncoder(os.Stderr).Encode(Config{"you@example.com", "TOKENHERE1234"})
		if err != nil {
			fmt.Println(err) // This is impossible, but errcheck lint. 😳
		}
	}
	flag.Parse()

	switch *format {
	case "plain":
	case "table":
	case "json":
	default:
		fmt.Fprintln(os.Stderr, "could not use specified format", *format)
		return
	}

	if len(os.Args) == 1 {
		flag.Usage()
		return
	}

	_, token, err := getCreds()
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not read config", err)
		return
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)

	client := dnsimple.NewClient(tc)
	whoamiResponse, err := client.Identity.Whoami()

	if err != nil {
		fmt.Fprintln(os.Stderr, "could not connect to dnsimple", err)
		return
	}

	if whoamiResponse.Data.Account == nil {
		fmt.Fprintln(os.Stderr, "you need to use account token instead of user token")
		return
	}
	accountID := strconv.FormatInt(whoamiResponse.Data.Account.ID, 10)

	if *list {
		domainsResponse, err := client.Domains.ListDomains(accountID, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get domains %v\n", err)
			return
		}
		for _, domain := range domainsResponse.Data {
			if *verbose {
				fmt.Println(domain.Name, domain.ExpiresOn)
			} else {
				fmt.Println(domain.Name)
			}
		}
		return
	}
	if *update != "" {
		id, err := createOrUpdate(client, *update, accountID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not create or update:", err)
		} else {
			fmt.Printf("record written with id %s\n", id)
		}
		return
	}
	if *del != "" {
		id, err := deleteRecord(client, *del, accountID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not delete:", err)
		} else {
			fmt.Printf("record deleted with id %s\n", id)
		}
		return
	}
	if len(flag.Args()) == 0 {
		flag.Usage()
		return
	}
	options := &dnsimple.ZoneRecordListOptions{}
	if *typeR != `` {
		options.Type = *typeR
	}
	records, err := listRecords(client, accountID, flag.Args()[0], options)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not get records:", err)
	}
	if *verbose {
		log.Println("found ", len(records), " records")
	}
	sort.Sort(records)

	err = FormatZoneRecords(records, *format)
	if err != nil {
		log.Println("error: err")
	}
}

// listRecords pages through records
func listRecords(client *dnsimple.Client, accountID, domain string,
	options *dnsimple.ZoneRecordListOptions) (records zoneRecords, err error) {
	if options == nil {
		options = &dnsimple.ZoneRecordListOptions{}
	}
	for p := 1; ; p++ {
		listZoneRecordsResponse, err := client.Zones.ListRecords(accountID, domain, options)
		if err != nil {
			return nil, err
		}
		for i := range listZoneRecordsResponse.Data {
			records = append(records, listZoneRecordsResponse.Data[i])
		}
		if options.Page == 0 {
			options.Page = 2
		} else {
			options.Page++
		}
		if p >= listZoneRecordsResponse.Pagination.TotalPages {
			break
		}
	}
	return
}

// FormatZoneRecords takes a slice of dnsimple.ZoneRecord and formats it in the specified format.
func FormatZoneRecords(zones []dnsimple.ZoneRecord, format string) error {
	if format == "json" {
		err := json.NewEncoder(os.Stdout).Encode(zones)
		if err != nil {
			return err
		}
		return nil
	}
	if format == "table" {
		fmt.Printf("+-%-30s-+-%-5s-+-%-7s-+-%-30s-+\n", strings.Repeat("-", 30), "-----", "-------", strings.Repeat("-", 30))
		fmt.Printf("| %-30s | %-5s | %-7s | %-30s |\n", "Name", "Type", "TTL", "Content")
		fmt.Printf("+-%-30s-+-%-5s-+-%-7s-+-%-30s-+\n", strings.Repeat("-", 30), "-----", "-------", strings.Repeat("-", 30))
	}
	for _, zone := range zones {
		if zone.Name == `` {
			zone.Name = `.`
		}
		switch format {
		case "plain":
			if *verbose {
				fmt.Printf("%s %s %s %d (%d) %s %v\n", zone.ZoneID, zone.Name, zone.Type, zone.Priority, zone.TTL, zone.Content, zone.UpdatedAt)
			} else if flag.NFlag() > 1 {
				fmt.Printf("%s %s %s (%d) %s\n", zone.ZoneID, zone.Name, zone.Type, zone.TTL, zone.Content)
			} else {
				fmt.Printf("%s %s (%d) %s\n", zone.Name, zone.Type, zone.TTL, zone.Content)

			}
		case "table":
			fmt.Printf("| %-30s | %-5s | %7d | %-30s |\n", zone.Name, zone.Type, zone.TTL, zone.Content)
		default:
			return fmt.Errorf("invalid format %v", format)
		}
	}
	if format == "table" {
		fmt.Printf("+-%-30s-+-%-5s-+-%-7s-+-%-30s-+\n", strings.Repeat("-", 30), "-----", "-------", strings.Repeat("-", 30))

	}
	return nil
}

type zoneRecords []dnsimple.ZoneRecord

func (z zoneRecords) Len() int           { return len(z) }
func (z zoneRecords) Less(i, j int) bool { return z[i].Name < z[j].Name }
func (z zoneRecords) Swap(i, j int)      { z[i], z[j] = z[j], z[i] }

var configFileName = func() string {
	if os.Getenv("DOMASIMU_CONF") != "" {
		return os.Getenv("DOMASIMU_CONF")
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Domasimu", "config")
	case "darwin":
		f := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Domasimu", "config")
		fh, err := os.Open(f)
		if err == nil {
			fh.Close()
			return f
		}
	}
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		f := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "domasimu", "config")
		fh, err := os.Open(f)
		if err == nil {
			fh.Close()
			return f
		}
	}
	f := filepath.Join(os.Getenv("HOME"), ".config", "domasimu", "config")
	fh, err := os.Open(f)
	if err == nil {
		fh.Close()
		return f
	}
	return filepath.Join(os.Getenv("HOME"), ".domasimurc")
}()

func getCreds() (string, string, error) {
	var config Config
	_, err := toml.DecodeFile(configFileName, &config)
	if err != nil {
		return "", "", err
	}
	return config.User, config.Token, nil
}

// Config represents the user and token config for dnsimple.
type Config struct {
	User  string
	Token string
}

func createOrUpdate(client *dnsimple.Client, message string, accountID string) (string, error) {
	pieces := strings.Split(message, " ")
	if len(pieces) != 6 {
		return "", fmt.Errorf("expected space seperated domain, name, type, oldvalue, newvalue, ttl")
	}

	domain := pieces[0]
	changeRecord := dnsimple.ZoneRecord{
		Name: pieces[1],
		Type: pieces[2],
	}
	oldValue := pieces[3]
	newRecord := changeRecord
	newRecord.Content = pieces[4]
	if *value != "" {
		newRecord.Content = *value
	}
	ttl, err := strconv.Atoi(pieces[5])
	if err != nil {
		return "", fmt.Errorf("could not convert %s to int: %w", pieces[5], err)
	}
	newRecord.TTL = ttl
	id, err := getRecordIDByValue(client, domain, oldValue, accountID, &changeRecord)
	if err != nil {
		return "", err
	}

	var respID string
	if id == 0 {
		zoneRecordResponse, err := client.Zones.CreateRecord(accountID, domain, newRecord)
		if err != nil {
			return "", err
		}
		respID = strconv.FormatInt(zoneRecordResponse.Data.ID, 10)
		if err != nil {
			return "", err
		}
	} else {
		zoneRecordResponse, err := client.Zones.UpdateRecord(accountID, domain, id, newRecord)
		if err != nil {
			return "", err
		}
		respID = strconv.FormatInt(zoneRecordResponse.Data.ID, 10)
	}

	return respID, nil
}

func deleteRecord(client *dnsimple.Client, message, accountID string) (string, error) {
	pieces := strings.Split(message, " ")
	if len(pieces) != 4 {
		return "", fmt.Errorf("expected space seperated domain, name, type, value")
	}
	domain := pieces[0]
	changeRecord := dnsimple.ZoneRecord{
		Name: pieces[1],
		Type: pieces[2],
	}
	value := pieces[3]
	id, err := getRecordIDByValue(client, domain, value, accountID, &changeRecord)
	if err != nil {
		return "", err
	}
	if id == 0 {
		return "", fmt.Errorf("could not find record")
	}
	_, err = client.Zones.DeleteRecord(accountID, domain, id)
	respID := strconv.FormatInt(id, 10)

	return respID, err
}

func getRecordIDByValue(client *dnsimple.Client, domain, value, accountID string, changeRecord *dnsimple.ZoneRecord) (int64, error) {
	options := &dnsimple.ZoneRecordListOptions{}
	if changeRecord.Type != "" {
		options.Type = changeRecord.Type
	}
	recordResponse, err := listRecords(client, accountID, domain, options)
	if err != nil {
		return 0, err
	}
	var id int64
	for _, record := range recordResponse {
		if record.Name == changeRecord.Name && record.Type == changeRecord.Type && record.Content == value {
			id = record.ID
			break
		}
	}
	return id, nil
}
