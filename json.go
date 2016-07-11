package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

func Encode(value interface{}) string {
	return string(EncodeBytes(value))
}

func EncodeBytes(value interface{}) []byte {
	var b []byte
	var err error

	b, err = json.MarshalIndent(value, "", "  ")

	if err != nil {
		fmt.Printf("%v", err)
	}
	return b
}

func EncodeBuffer(value interface{}) *bytes.Buffer {
	return bytes.NewBuffer(EncodeBytes(value))
}

func Decode(body io.ReadCloser, v interface{}) error {
	content, err := ioutil.ReadAll(body)
	body.Close()
	if err != nil {
		return err
	}

	// fmt.Printf("Insight %v\n", string(content))
	err = json.Unmarshal(content, v)

	if err != nil {
		return err
	}
	return nil
}

func DecodeBytes(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	return nil
}

func DecodeBuffer(buf *bytes.Buffer, v interface{}) error {
	err := json.Unmarshal(buf.Bytes(), v)
	if err != nil {
		return err
	}
	return nil
}
