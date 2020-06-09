package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver"
)

type PypiData struct {
	Version  string
	Requires string
	Releases []string
}

func main() {
	pypi_url := "https://pypi.org/pypi/pulpcore/json"
	pulp_plugins := []string{
		"galaxy-ng",
		"pulp-ansible",
		"pulp-certguard",
		"pulp-container",
		"pulp-cookbook",
		"pulp-deb",
		"pulp-file",
		"pulp-gem",
		"pulp-maven",
		"pulp-npm",
		"pulp-python",
		"pulp-rpm",
	}
	pulpcore_data := getPypiData(pypi_url)
	for index := range pulpcore_data.Releases {
		pulpcore_version := pulpcore_data.Releases[len(pulpcore_data.Releases)-1-index]
		if strings.Contains(pulpcore_version, "3.0.0") {
			// avoiding rc versions
			printCompatiblePlugins(pypi_url, "3.0.0", pulp_plugins)
			break
		}
		printCompatiblePlugins(pypi_url, pulpcore_version, pulp_plugins)
	}
}

func getPypiData(url string) *PypiData {
	var myClient = &http.Client{Timeout: 10 * time.Second}
	r, err := myClient.Get(url)
	if err != nil {
		panic(err.Error())
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	var data interface{}
	e := json.Unmarshal(body, &data)
	if e != nil {
		panic(err.Error())
	}

	itemsMap := data.(map[string]interface{})
	info := itemsMap["info"].(map[string]interface{})
	version := fmt.Sprintf("%v", info["version"])
	releasesMap := itemsMap["releases"].(map[string]interface{})
	releasesInterface := reflect.ValueOf(releasesMap).MapKeys()
	releases := make([]string, len(releasesInterface))
	for i, v := range releasesInterface {
		releases[i] = v.String()
	}
	sort.Strings(releases)

	requiresInterface := info["requires_dist"].([]interface{})
	requires := ""
	for _, v := range requiresInterface {
		if strings.Contains(v.(string), "pulpcore") {
			requires = strings.Replace(v.(string), "(", "", -1)
			requires = strings.Replace(requires, ")", "", -1)
		}
	}

	return &PypiData{version, requires, releases}
}

func printCompatiblePlugins(pypi_url string, pulpcore_version string, plugins []string) {
	shown := false
	for index, plugin := range plugins {
		if plugin == "remove" {
			continue
		}
		pypi_data := getPypiData(strings.Replace(pypi_url, "pulpcore", plugin, -1))
		req := strings.Fields(pypi_data.Requires)
		c, err := semver.NewConstraint(strings.Replace(req[1], "~=", "~", -1))
		if err != nil {
			c, _ = semver.NewConstraint("<3.0.1")
		}
		v, err := semver.NewVersion(pulpcore_version)
		if err != nil {
			panic(err.Error())
		}
		if c.Check(v) {
			if !shown {
				fmt.Println(fmt.Sprintf("\nCompatible with pulpcore-%s", pulpcore_version))
				shown = true
			}
			fmt.Println(fmt.Sprintf(" -> %s-%s requirement: %s", plugin, pypi_data.Version, pypi_data.Requires))
			plugins[index] = "remove"
		}
	}
}
