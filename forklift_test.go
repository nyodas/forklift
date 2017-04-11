package main

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func TestLoadConfigThrowError(t *testing.T) {
	_, err := loadConfig("")
	if err == nil {
		t.Error("Path wasn't set but error wasn't thrown")
	}
}

func TestLoadConfigNoErr(t *testing.T) {
	d1 := []byte("test_load_config")
	_ = ioutil.WriteFile("/tmp/.test", d1, 0644)

	fileContent, err := loadConfig("/tmp/.test")
	if err != nil && string(fileContent) == string(d1) && reflect.DeepEqual(fileContent, d1) {
		t.Error("File is present there shouldn't be any errors")
	}
}
