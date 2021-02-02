package database

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	log.SetFormatter(&log.JSONFormatter{})
	os.Exit(m.Run())
}

func TestCreateDatabase(t *testing.T) {
	_, err := NewDatabaseConnection(NewSQLiteConnector())

	if err != nil {
		t.Error(err.Error())
	}

}
