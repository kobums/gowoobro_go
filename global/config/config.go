package config

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type DatabaseType int

const (
	_ DatabaseType = iota
	Mysql
	Postgresql
	Sqlserver
)

type _Tls struct {
	Use  bool   `yaml:"use"`
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}
type _Mode struct {
	Id           int       `yaml:"id"`
	Port         string    `yaml:"port"`
	Tls          _Tls      `yaml:"tls"`
	UploadPath   string    `yaml:"path"`
	DocumentRoot string    `yaml:"documentRoot"`
	Mail         _Mail     `yaml:"mail"`
	Sms          _Sms      `yaml:"sms"`
	Cors         []string  `yaml:"cors"`
	Server       []string  `yaml:"server"`
	Database     _Database `yaml:"database"`
	Log          _Log      `yaml:"log"`
}

type _Database struct {
	Host             string       `yaml:"host"`
	Port             string       `yaml:"port"`
	Name             string       `yaml:"name"`
	Owner            string       `yaml:"owner"`
	User             string       `yaml:"user"`
	Password         string       `yaml:"password"`
	Type             DatabaseType `yaml:"typeInner"`
	TypeString       string       `yaml:"type"`
	ConnectionString string       `yaml:"connectionString"`
}

type _Mail struct {
	Sender string `yaml:"sender"`
}

type _Sms struct {
	User   string `yaml:"user"`
	Key    string `yaml:"key"`
	Sender string `yaml:"sender"`
}

type _Log struct {
	Level    string `yaml:"level"`
	Console  bool   `yaml:"console"`
	Web      bool   `yaml:"web"`
	Database bool   `yaml:"database"`
	File     string `yaml:"file"`
	Limit    struct {
		Size  int `yaml:"size"`
		Count int `yaml:"count"`
		Days  int `yaml:"days"`
	} `yaml:"limit"`
}

type Config struct {
	Version    string `yaml:"version"`
	Develop    _Mode  `yaml:"develop"`
	Production _Mode  `yaml:"production"`
}

var Mail _Mail
var Database _Database
var Sms _Sms
var Tls _Tls
var Log _Log
var UploadPath string
var DocumentRoot string
var Version string
var Mode string
var Port string
var Cors []string
var Server []string
var _value map[string]interface{}

var CrawlerId string

func Init() {
	config := &Config{}
	obj := make(map[string]interface{})

	buf, err := os.ReadFile(".env.yml")
	if err == nil {
		err = yaml.Unmarshal(buf, config)
		if err != nil {
			log.Println(err.Error())
		} else {
			err = yaml.Unmarshal(buf, obj)
			if err != nil {
				log.Println(err.Error())
			}
		}
	}

	Tls.Use = false

	Mode = os.Getenv("APP_MODE")

	if len(os.Args) == 3 {
		if os.Args[1] == "--mode" {
			Mode = os.Args[2]
		}
	}

	if Mode != "production" {
		Mode = "develop"
	}

	Log.Level = "debug"
	Log.Console = true
	Log.Web = true
	Log.Database = true
	Log.File = "webdata/log/system.log"

	if Mode == "production" {
		Mail = config.Production.Mail
		Sms = config.Production.Sms
		UploadPath = config.Production.UploadPath
		DocumentRoot = config.Production.DocumentRoot
		Port = config.Production.Port
		Database = config.Production.Database
		Cors = config.Production.Cors
		Log = config.Production.Log
		Server = config.Production.Server
		Tls = config.Production.Tls

		if _, exist := obj["production"]; exist {
			_value = obj["production"].(map[string]interface{})
		}
	} else {
		Mail = config.Develop.Mail
		Sms = config.Develop.Sms
		UploadPath = config.Develop.UploadPath
		DocumentRoot = config.Develop.DocumentRoot
		Port = config.Develop.Port
		Database = config.Develop.Database
		Cors = config.Develop.Cors
		Log = config.Develop.Log
		Server = config.Develop.Server
		Tls = config.Develop.Tls

		if _, exist := obj["develop"]; exist {
			_value = obj["develop"].(map[string]interface{})
		}
	}

	if DocumentRoot == "" {
		DocumentRoot = "dist"
	}

	envPort := os.Getenv("PORT")
	if envPort != "" {
		Port = envPort
	}

	envLogLevel := os.Getenv("LOG_LEVEL")
	envLogConsole := os.Getenv("LOG_CONSOLE")
	envLogWeb := os.Getenv("LOG_WEB")
	envLogDatabase := os.Getenv("LOG_DATABASE")
	envLogFile := os.Getenv("LOG_FILE")
	envLogDays := os.Getenv("LOG_DAYS")
	if envLogLevel != "" {
		Log.Level = envLogLevel
	}
	if envLogConsole != "" {
		if envLogConsole == "Y" {
			Log.Console = true
		}
	}
	if envLogWeb != "" {
		if envLogWeb == "Y" {
			Log.Web = true
		}
	}
	if envLogDatabase != "" {
		if envLogDatabase == "Y" {
			Log.Database = true
		}
	}
	if envLogFile != "" {
		Log.File = envLogFile
	}
	if envLogDays != "" {
		days, _ := strconv.Atoi(envLogDays)
		if days == 0 {
			Log.File = ""
		}
		Log.Limit.Days = days
	}

	Log.Level = "debug"
	Log.Console = true
	Log.Web = true
	Log.Database = true
	Log.File = "webdata/log/system.log"
	envDBType := os.Getenv("DB_TYPE")
	envDBHost := os.Getenv("DB_HOST")
	envDBPort := os.Getenv("DB_PORT")
	envDBName := os.Getenv("DB_NAME")
	envDBUser := os.Getenv("DB_USER")
	envDBPass := os.Getenv("DB_PASS")
	if envDBType != "" {
		Database.TypeString = envDBType
	}
	if envDBHost != "" {
		Database.Host = envDBHost
	}
	if envDBPort != "" {
		Database.Port = envDBPort
	}
	if envDBName != "" {
		Database.Name = envDBName
	}
	if envDBUser != "" {
		Database.User = envDBUser
	}
	if envDBPass != "" {
		Database.Password = envDBPass
	}

	envTlsCert := strings.ToUpper(os.Getenv("TLS_CERT"))
	if envTlsCert != "" {
		Tls.Cert = envTlsCert
	}
	envTlsKey := strings.ToUpper(os.Getenv("TLS_KEY"))
	if envTlsKey != "" {
		Tls.Key = envTlsKey
	}
	envTlsUse := strings.ToUpper(os.Getenv("TLS_USE"))
	if envTlsUse == "TRUE" || envTlsUse == "T" || envTlsUse == "YES" || envTlsUse == "Y" {
		Tls.Use = true
		if Tls.Cert == "" {
			Tls.Cert = path.Join(UploadPath + "certs/ssl.crt")
		}
		if Tls.Key == "" {
			Tls.Key = path.Join(UploadPath + "certs/ssl.key")
		}
	}

	if Port == "" {
		Port = "80"
	}

	if UploadPath == "" {
		UploadPath = "webdata"
	}

	if Database.TypeString == "postgres" || Database.TypeString == "postgresql" {
		if Database.Port == "" {
			Database.Port = "5432"
		}

		Database.Type = Postgresql
	} else if Database.TypeString == "sqlserver" || Database.TypeString == "mssql" {
		if Database.Port == "" {
			Database.Port = "1433"
		}

		Database.Type = Sqlserver
	} else {
		if Database.Port == "" {
			Database.Port = "3306"
		}

		Database.Type = Mysql
	}

	if Database.ConnectionString == "" {
		if Database.Type == Postgresql {
			Database.ConnectionString = fmt.Sprintf("host=%v port=%v user=%v password=%v dbname=%v sslmode=disable", Database.Host, Database.Port, Database.User, Database.Password, Database.Name)
		} else if Database.Type == Sqlserver {
			Database.ConnectionString = fmt.Sprintf("server=%v;port=%v;user id=%v,password=%v;database=%v", Database.Host, Database.Port, Database.User, Database.Password, Database.Name)
		} else {
			Database.ConnectionString = fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", Database.User, Database.Password, Database.Host, Database.Port, Database.Name)
		}
	}

	Version = config.Version
	CrawlerId = "chin1525"
}

func Get(name string) interface{} {
	return _value[name]
}

func GetString(name string) string {
	return _value[name].(string)
}

func GetInt(name string) int {
	return _value[name].(int)
}
