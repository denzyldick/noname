package main

import "fmt"

type Config struct {
	port     int
	host     string
	imageUrl string
	endpoint string
}

func (config *Config) getAddress() string {
	return fmt.Sprintf("%s:%d", config.host, config.port)
}

func (config *Config) getEndpoint() string {
	return config.endpoint
}
