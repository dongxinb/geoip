package maxmind

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
)

const (
	typeCountryCSV = "ipinfoCSV"
	descCountryCSV = "Convert MaxMind ipinfo country/continent CSV data to other formats"
)

var (
	defaultFile = filepath.Join("./", "ipinfo", "country.csv")
)

func init() {
	lib.RegisterInputConfigCreator(typeCountryCSV, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newipinfo2CountryCSV(action, data)
	})
	lib.RegisterInputConverter(typeCountryCSV, &ipinfo2CountryCSV{
		Description: descCountryCSV,
	})
}

func newipinfo2CountryCSV(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
	var tmp struct {
		File       string     `json:"file"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.File == "" {
		tmp.File = defaultFile
	}

	return &ipinfo2CountryCSV{
		Type:        typeCountryCSV,
		Action:      action,
		Description: descCountryCSV,
		File:        tmp.File,
		Want:        tmp.Want,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type ipinfo2CountryCSV struct {
	Type        string
	Action      lib.Action
	Description string
	File        string
	Want        []string
	OnlyIPType  lib.IPType
}

func (g *ipinfo2CountryCSV) GetType() string {
	return g.Type
}

func (g *ipinfo2CountryCSV) GetAction() lib.Action {
	return g.Action
}

func (g *ipinfo2CountryCSV) GetDescription() string {
	return g.Description
}

func (g *ipinfo2CountryCSV) Input(container lib.Container) (lib.Container, error) {

	entries := make(map[string]*lib.Entry)

	if g.File != "" {
		if err := g.process(g.File, entries); err != nil {
			return nil, err
		}
	}

	var ignoreIPType lib.IgnoreIPOption
	switch g.OnlyIPType {
	case lib.IPv4:
		ignoreIPType = lib.IgnoreIPv6
	case lib.IPv6:
		ignoreIPType = lib.IgnoreIPv4
	}

	for name, entry := range entries {
		switch g.Action {
		case lib.ActionAdd:
			if err := container.Add(entry, ignoreIPType); err != nil {
				return nil, err
			}
		case lib.ActionRemove:
			container.Remove(name, ignoreIPType)
		default:
			return nil, lib.ErrUnknownAction
		}
	}

	return container, nil
}

func (g *ipinfo2CountryCSV) process(file string, entries map[string]*lib.Entry) error {
	print("process ipinfo")
	if entries == nil {
		entries = make(map[string]*lib.Entry)
	}

	fReader, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fReader.Close()

	reader := csv.NewReader(fReader)
	lines, err := reader.ReadAll()
	if err != nil {
		return err
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range g.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	for _, line := range lines[1:] {
		startIp := strings.ToLower(strings.TrimSpace(line[0]))
		toIp := strings.ToLower(strings.TrimSpace(line[1]))
		country := strings.ToUpper(strings.TrimSpace(line[2]))
		continent := strings.ToUpper(strings.TrimSpace(line[4]))
		// special case: CN
		if country == "CN" {
			continent = country
		}
		if country == "SG" || country == "MY" || country == "ID" || country == "TH" {
			continent = "SG"
		}
		if len(wantList) > 0 {
			if _, found := wantList[continent]; !found {
				continue
			}
		}
		entry, found := entries[continent]
		if !found {
			entry = lib.NewEntry(continent)
		}
		if err := entry.AddRange(startIp, toIp); err != nil {
			return err
		}
		entries[continent] = entry
	}

	return nil
}
