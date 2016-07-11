package main

import "time"

type config struct {
	State string

	ReportingUrl string
	AuthorizeUrl string
	RedirectUri  string
	TokenUrl     string

	ClientId     string
	ClientSecret string
	Scope        string

	AppId string

	DataPath   string
	TokensPath string
	ExportPath string

	FirstDate time.Time

	TestQuery string
}

var Config = config{
	State: "12345678",

	ReportingUrl: "https://www.googleapis.com/analytics/v3/data/ga",
	AuthorizeUrl: "https://accounts.google.com/o/oauth2/v2/auth",
	RedirectUri:  "http://localhost:8080/redirect",
	TokenUrl:     "https://www.googleapis.com/oauth2/v3/tokeninfo",

	ClientId:     "992587431255-fp42a6812ja9h0e1bf4cc54pb4kn674r.apps.googleusercontent.com",
	ClientSecret: "EkNpkuy9MLH8DCAkk23d8_l1",
	Scope:        "https://www.googleapis.com/auth/analytics",

	AppId: "103839101",

	DataPath:   "./data",
	TokensPath: "./data/tokens",
	ExportPath: "./exports",

	FirstDate: time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC),

	TestQuery: "?ids=ga:%v&access_token=%v&start-date=%v&end-date=%v&metrics=ga:pageviews&dimensions=ga:browser,ga:operatingSystem,ga:country,ga:city,ga:networkLocation,ga:language",
}
