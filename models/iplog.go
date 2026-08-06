package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gowoobro/global/config"

	log "gowoobro/global/log"
)

// Iplog 는 방문 기록 한 건이다. 예전에 ipblock_tb 가 하던 "방문한 사용자의 IP 를
// 그대로 쌓아두는" 역할을 iplog_tb 가 이어받는다. ipblock_tb 는 차단 목록 전용.
//
// 다른 모델과 달리 buildtool 생성이 아니라 손으로 쓴 최소 구현이다. 적재(Insert)만
// 필요해서 조회·수정 계열은 만들지 않았다. 조회는 ipblock_tb 처럼 DB 에서 직접 한다.
type Iplog struct {
	Id      int64  `json:"id"`
	Address string `json:"address"`
	Path    string `json:"path"`
	Agent   string `json:"agent"`
	Os      string `json:"os"`
	Browser string `json:"browser"`
	Date    string `json:"date"`
}

type IplogManager struct {
	Conn   *Connection
	Result *sql.Result
	Log    bool
}

func NewIplogManager(conn *Connection) *IplogManager {
	var item IplogManager

	if conn == nil {
		item.Conn = NewConnection()
	} else {
		item.Conn = conn
	}

	item.Log = config.Log.Database

	return &item
}

func (p *IplogManager) Close() {
	if p.Conn != nil {
		p.Conn.Close()
	}
}

func (p *IplogManager) Insert(item *Iplog) error {
	if !p.Conn.IsConnect() {
		return errors.New("Connection Error")
	}

	if item.Date == "" {
		t := time.Now().UTC().Add(time.Hour * 9)
		item.Date = fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	}

	// 컬럼 길이를 넘치면 insert 자체가 실패한다. User-Agent 와 그 파생값은
	// 클라이언트가 마음대로 보내는 값이라 길이를 믿을 수 없다.
	if len(item.Path) > 255 {
		item.Path = item.Path[:255]
	}
	if len(item.Agent) > 255 {
		item.Agent = item.Agent[:255]
	}
	if len(item.Os) > 50 {
		item.Os = item.Os[:50]
	}
	if len(item.Browser) > 50 {
		item.Browser = item.Browser[:50]
	}

	query := "insert into iplog_tb (il_address, il_path, il_agent, il_os, il_browser, il_date) values (?, ?, ?, ?, ?, ?)"

	if p.Log {
		log.Debug().Str("query", query).Any("param", []interface{}{item.Address, item.Path, item.Agent, item.Os, item.Browser, item.Date}).Msg("SQL")
	}

	res, err := p.Conn.Exec(query, item.Address, item.Path, item.Agent, item.Os, item.Browser, item.Date)
	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
		p.Result = nil
		return err
	}

	p.Result = &res
	return nil
}
