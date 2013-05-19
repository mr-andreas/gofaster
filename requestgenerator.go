package main

import "net/http"

type RequestGenerator interface {
	GenerateRequest() (*http.Request, error)
}

type StaticRequestGenerator struct {
	Url string
}

func (s *StaticRequestGenerator) GenerateRequest() (*http.Request, error) {
	return http.NewRequest("GET", "http://www.mostphotos.com", nil)
}
