package pqueue

import "testing"

func TestGetConfig(t *testing.T) {
	dsn := GetConfig("psql_dsn")
	if dsn != "host=localhost user=postgres dbname=postgres sslmode=disable" {
		t.Error("unmatch psql dsn")
	}
}

func TestGetEmptyConfig(t *testing.T) {
	empty := GetConfig("empty")
	if empty != "" {
		t.Errorf("expect empty string, actual %s", empty)
	}
}

func TestSetConfig(t *testing.T) {
	testStr := "set configuration string"
	SetConfig("test", testStr)
	str := GetConfig("test")
	if str != testStr {
		t.Errorf("expect %s, actual %s", testStr, str)
	}
}
