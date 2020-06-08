package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
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
	fmt.Println("Latest pulpcore version:", pulpcore_data.Version)
	for _, plugin := range pulp_plugins {
		pypi_data := getPypiData(strings.Replace(pypi_url, "pulpcore", plugin, -1))
		req := strings.Fields(pypi_data.Requires)
		c, err := semver.NewConstraint(req[1])
		if err != nil {
			c, _ = semver.NewConstraint("<3.0.1")
		}
		v, err := semver.NewVersion(pulpcore_data.Version)
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(fmt.Sprintf("%s-%s requires: %s is compatible:", plugin, pypi_data.Version, pypi_data.Requires), c.Check(v))
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
