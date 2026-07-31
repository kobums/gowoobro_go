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

// 관리자 인증. 둘 다 환경변수로만 받는다 — 기본값을 두면 그 값을 아는 사람이
// 누구나 관리자가 되므로, 비어 있으면 로그인 자체를 거부한다(router/auth.go).
var JwtSecret string
var AdminPassword string

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

	// 업로드 경로도 .env.yml 에만 있던 값이다. 운영에서는 마운트된 볼륨
	// (/usr/local/main/webdata) 을 가리켜야 한다. 이게 비어 상대경로 "webdata"
	// 로 떨어지면 업로드 파일이 컨테이너 안에만 쌓였다가 재배포 때 사라진다.
	if envUploadPath := os.Getenv("UPLOAD_PATH"); envUploadPath != "" {
		UploadPath = envUploadPath
	}

	if envDocumentRoot := os.Getenv("DOCUMENT_ROOT"); envDocumentRoot != "" {
		DocumentRoot = envDocumentRoot
	}

	// CORS 는 원래 .env.yml 에만 있어서, 그 파일 하나 때문에 이미지에 설정을
	// 구워 넣어야 했다. 프론트(gowoobro.com)와 API(gowoobroapi.gowoobro.com)가
	// 서로 다른 오리진이라 이 목록이 비면 CORS 미들웨어가 아예 안 붙고
	// (services/http.go) 브라우저가 API 호출을 전부 막는다.
	//
	// tomelater/tomelatergo/config.go 와 같은 방식으로 콤마 구분 환경변수를 받는다.
	//   CORS=https://gowoobro.com,https://www.gowoobro.com
	if envCors := os.Getenv("CORS"); envCors != "" {
		Cors = nil
		for _, site := range strings.Split(envCors, ",") {
			if site = strings.TrimSpace(site); site != "" {
				Cors = append(Cors, site)
			}
		}
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

		// TypeString 은 sql.Open 에 그대로 넘어가는 드라이버 이름이다(models/db.go).
		// 이 분기는 이미 MySQL 로 판정했는데 TypeString 이 비어 있으면
		// sql.Open("") 이 되어 `unknown driver ""` 로 죽는다. .env.yml 을 쓰던
		// 시절엔 YAML 에 type 이 적혀 있어 드러나지 않던 구멍이다.
		if Database.TypeString == "" {
			Database.TypeString = "mysql"
		}
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

	JwtSecret = os.Getenv("JWT_SECRET")
	AdminPassword = os.Getenv("ADMIN_PASSWORD")

	// 여기서 기동을 막지는 않는다. 이 사이트는 대부분이 공개 페이지라, 관리자
	// 인증 설정이 빠졌다고 공개 사이트까지 죽이는 건 과하다. 대신 로그로 크게
	// 알리고 로그인만 거부한다.
	if JwtSecret == "" {
		log.Println("[warn] JWT_SECRET 이 없다. 관리자 로그인이 비활성화된다.")
	}
	if AdminPassword == "" {
		log.Println("[warn] ADMIN_PASSWORD 가 없다. 관리자 로그인이 비활성화된다.")
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
