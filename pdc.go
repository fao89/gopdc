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

// PypiData : stores the desired PyPI data
type PypiData struct {
	Name     string
	Version  string
	Requires string
	Releases []string
}

func main() {
	pypiURL := "https://pypi.org/pypi/pulpcore/json"
	pulpPlugins := []string{
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
	channel := make(chan PypiData)
	go getPypiData(pypiURL, channel)
	pulpcoreData := <-channel

	for _, plugin := range pulpPlugins {
		go getPypiData(strings.Replace(pypiURL, "pulpcore", plugin, -1), channel)
	}
	pulpPluginsData := make([]PypiData, len(pulpPlugins))
	for index := range pulpPlugins {
		pulpPluginsData[index] = <-channel
	}
	for index := range pulpcoreData.Releases {
		pulpcoreVersion := pulpcoreData.Releases[len(pulpcoreData.Releases)-1-index]
		if strings.Contains(pulpcoreVersion, "3.0.0") {
			// avoiding rc versions
			printCompatiblePlugins(pypiURL, "3.0.0", pulpPluginsData)
			break
		}
		printCompatiblePlugins(pypiURL, pulpcoreVersion, pulpPluginsData)
	}
}

func getPypiData(url string, channel chan PypiData) {
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
	name := fmt.Sprintf("%v", info["name"])
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

	channel <- PypiData{name, version, requires, releases}
}

func printCompatiblePlugins(pypiURL string, pulpcoreVersion string, plugins []PypiData) {
	shown := false
	for index, pypiData := range plugins {
		if pypiData.Name == "remove" {
			continue
		}
		req := strings.Fields(pypiData.Requires)
		c, err := semver.NewConstraint(strings.Replace(req[1], "~=", "~", -1))
		if err != nil {
			c, _ = semver.NewConstraint("<3.0.1")
		}
		v, err := semver.NewVersion(pulpcoreVersion)
		if err != nil {
			panic(err.Error())
		}
		if c.Check(v) {
			if !shown {
				fmt.Println(fmt.Sprintf("\nCompatible with pulpcore-%s", pulpcoreVersion))
				shown = true
			}
			fmt.Println(fmt.Sprintf(" -> %s-%s requirement: %s", pypiData.Name, pypiData.Version, pypiData.Requires))
			pypiData.Name = "remove"
			plugins[index] = pypiData
		}
	}
}
