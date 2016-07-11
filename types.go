package main

import "time"

type Date struct {
	Date time.Time `json:"date"`
}

type GAColumnHeader struct {
	Name       string `json:"name"`
	ColumnType string `json:"columnType"`
	DataType   string `json:"dataType"`
}

type GAResponse struct {
	NextLink      string           `json:"nextLink"`
	ColumnHeaders []GAColumnHeader `json:"columnHeaders"`
	Rows          [][]string       `json:"rows"`
}
