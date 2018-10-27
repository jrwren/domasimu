package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dnsimple/dnsimple-go/dnsimple"
)

// FormatZoneRecord formats a dnsimple.ZoneRecord in the specified format, or returns an error when it failed.
func FormatZoneRecord(zone dnsimple.ZoneRecord, format string) (string, error) {
	switch format {
	case "plain":
		return fmt.Sprintf("%s %s (%d) %s\n", zone.Name, zone.Type, zone.TTL, zone.Content), nil
	case "json":
		enc, err := json.Marshal(zone)
		if err != nil {
			return "", err
		}
		return string(enc), nil
	case "table":
		return fmt.Sprintf("| %-30s | %-5s | %7s | %-30s |\n", zone.Name, zone.Type, zone.TTL, zone.Content), nil
	default:
		return "", fmt.Errorf("invalid format %v", format)
	}

}

// FormatZoneRecords takes a slice of dnsimple.ZoneRecord and formats it in the specified format.
func FormatZoneRecords(zones []dnsimple.ZoneRecord, format string) (string, error) {
	if format == "json" {
		enc, err := json.Marshal(zones)
		if err != nil {
			return "", err
		}
		return string(enc), nil
	}

	var ret string
	if format == "table" {
		ret = fmt.Sprintf("+-%-30s-+-%-5s-+-%-7s-+-%-30s-+\n", strings.Repeat("-", 30), "-----", "-------", strings.Repeat("-", 30))
		ret = fmt.Sprintf("%s| %-30s | %-5s | %-7s | %-30s |\n", ret, "Name", "Type", "TTL", "Content")
		ret = fmt.Sprintf("%s+-%-30s-+-%-5s-+-%-7s-+-%-30s-+\n", ret, strings.Repeat("-", 30), "-----", "-------", strings.Repeat("-", 30))
	}

	for _, zone := range zones {
		if formatted, err := FormatZoneRecord(zone, format); err != nil {
			return "", fmt.Errorf("error formatting zone %v: %v", zone, err)
		} else {
			ret = fmt.Sprintf("%s%s", ret, formatted)
		}
	}
	if format == "table" {
		ret = fmt.Sprintf("%s+-%-30s-+-%-5s-+-%-7s-+-%-30s-+\n", ret, strings.Repeat("-", 30), "-----", "-------", strings.Repeat("-", 30))
	}

	return ret, nil
}
