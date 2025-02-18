package proxy

import (
	"fmt"
)

// Proxy represents a configured proxy with its credentials
type Proxy struct {
	Host     string
	Port     string
	Username string
	Password string
}

// New creates a new Proxy instance with the given credentials
func New(host, port, username, password string) *Proxy {
	return &Proxy{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

// URL returns a proxy URL with the given country code appended to the username.
// The country code is appended in the format "username-country-XX" as required
// by BrightData's geolocation targeting feature.
func (p *Proxy) URL(countryCode string) (*string, error) {
	username := p.Username
	if countryCode != "" {
		username = fmt.Sprintf("%s-country-%s", p.Username, countryCode)
	}

	proxyURL := fmt.Sprintf("http://%s:%s@%s:%s", username, p.Password, p.Host, p.Port)
	return &proxyURL, nil
}
