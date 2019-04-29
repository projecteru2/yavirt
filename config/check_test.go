package config

import (
	"testing"

	"github.com/projecteru2/yavirt/test/assert"
)

type config struct {
	Debug        string
	Env          string `enum:"dev,test,uat,live"`
	ProfHTTPPort int    `toml:"prof_http_port" enum:"9999,8888"`
	Mysql        *mysql
}

type mysql struct {
	User string `range:"2-8"`
	Port int    `range:"6000-9999" json:"port"`
	Host string `json:"host"`
}

var newconf = func() *config {
	return &config{
		Mysql: &mysql{},
	}
}

func TestCheckNotFound(t *testing.T) {
	var conf = newconf()
	assert.NotNil(t, newChecker(conf, "NotFound").check())
	assert.NotNil(t, newChecker(conf, "Mysql.NotFound").check())
}

func TestCheckStrEnum(t *testing.T) {
	var conf = newconf()

	conf.Env = ""
	assert.NotNil(t, newChecker(conf, "Env").check())

	conf.Env = "unknown"
	assert.NotNil(t, newChecker(conf, "Env").check())

	conf.Env = "dev"
	assert.Nil(t, newChecker(conf, "Env").check())
}

func TestCheckStrRange(t *testing.T) {
	var conf = newconf()

	conf.Mysql.User = "a"
	assert.NotNil(t, newChecker(conf, "Mysql.User").check())

	conf.Mysql.User = string(make([]byte, 256))
	assert.NotNil(t, newChecker(conf, "Mysql.User").check())
}

func TestCheckIntEnum(t *testing.T) {
	var conf = newconf()
	conf.ProfHTTPPort = 80
	assert.NotNil(t, newChecker(conf, "ProfHTTPPort").check())

	conf.ProfHTTPPort = 8888
	assert.Nil(t, newChecker(conf, "ProfHTTPPort").check())
}

func TestCheckIntRange(t *testing.T) {
	var conf = newconf()
	conf.Mysql.Port = 3306
	assert.NotNil(t, newChecker(conf, "Mysql.Port").check())

	conf.Mysql.Port = 6606
	assert.Nil(t, newChecker(conf, "Mysql.Port").check())
}

func TestCheckNone(t *testing.T) {
	var conf = newconf()
	assert.Nil(t, newChecker(conf, "Debug").check())
	assert.Nil(t, newChecker(conf, "Mysql.Host").check())
}
