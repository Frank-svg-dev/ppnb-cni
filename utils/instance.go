package utils

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

type Metadata struct {
	UUID     string `json:"uuid"`
	Hostname string `json:"hostname"`
	Name     string `json:"name"`
}

func GetInstanceUUID() (instanceID string, hostname string) {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get("http://169.254.169.254/openstack/latest/meta_data.json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var meta Metadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		panic(err)
	}

	hostname, err = os.Hostname()
	if err != nil {
		panic(err)
	}
	
	return meta.UUID, hostname

}
