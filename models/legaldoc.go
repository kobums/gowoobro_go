package models

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gowoobro/global/config"
	log "gowoobro/global/log"
)

// Legaldoc 은 legaldoc_tb 의 한 행 — 어떤 프로젝트의 약관·개인정보 처리방침 등의
// 한 버전이다. 스키마와 규칙은 migrations/legaldoc_tb.sql 참고.
//
// 다른 모델과 달리 생성기 산출물이 아니라 손으로 쓴 것이라 필요한 메서드만 있다.
type Legaldoc struct {
	Id            int64  `json:"id"`
	Project       int64  `json:"project"`
	Type          string `json:"type"`
	Lang          string `json:"lang"`
	Version       int    `json:"version"`
	Title         string `json:"title"`
	Body          string `json:"body"`
	Effectivedate string `json:"effectivedate"`
	Date          string `json:"date"`
}

const legaldocColumns = "ld_id, ld_project, ld_type, ld_lang, ld_version, ld_title, ld_body, ld_effectivedate, ld_date"

// 문서 종류는 URL 경로 조각으로 그대로 쓰인다.
var legaldocTypePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var legaldocLangPattern = regexp.MustCompile(`^[a-z]{2}$`)

// LegaldocToday 는 시행일 비교에 쓰는 오늘 날짜(한국 시간, YYYY-MM-DD)다.
func LegaldocToday() string {
	return time.Now().UTC().Add(9 * time.Hour).Format("2006-01-02")
}

// Normalize 는 입력의 앞뒤 공백·대소문자를 정리한다. Validate 전에 부른다.
func (d *Legaldoc) Normalize() {
	d.Type = strings.ToLower(strings.TrimSpace(d.Type))
	d.Lang = strings.ToLower(strings.TrimSpace(d.Lang))
	d.Title = strings.TrimSpace(d.Title)
	d.Effectivedate = strings.TrimSpace(d.Effectivedate)
	if d.Lang == "" {
		d.Lang = "ko"
	}
}

// Validate 는 저장해도 되는 값인지 본다.
func (d *Legaldoc) Validate() error {
	if d.Project <= 0 {
		return errors.New("project is required")
	}
	if len(d.Type) > 30 || !legaldocTypePattern.MatchString(d.Type) {
		return errors.New("type must be lowercase letters, digits and hyphens (max 30)")
	}
	if !legaldocLangPattern.MatchString(d.Lang) {
		return errors.New("lang must be a two-letter code")
	}
	if d.Title == "" || utf8.RuneCountInString(d.Title) > 100 {
		return errors.New("title is required (max 100)")
	}
	if strings.TrimSpace(d.Body) == "" {
		return errors.New("body is required")
	}
	if t, err := time.Parse("2006-01-02", d.Effectivedate); err != nil || t.Format("2006-01-02") != d.Effectivedate {
		return errors.New("effectivedate must be YYYY-MM-DD")
	}
	return nil
}

type LegaldocManager struct {
	Conn *Connection
	Log  bool
}

func NewLegaldocManager(conn *Connection) *LegaldocManager {
	if conn == nil {
		conn = NewConnection()
	}
	return &LegaldocManager{Conn: conn, Log: config.Log.Database}
}

func (p *LegaldocManager) logError(err error) {
	if err != nil && p.Log {
		log.Error().Str("error", err.Error()).Msg("SQL")
	}
}

func (p *LegaldocManager) query(query string, params ...interface{}) []Legaldoc {
	items := make([]Legaldoc, 0)
	if p.Conn == nil || !p.Conn.IsConnect() {
		return items
	}

	if p.Log {
		log.Debug().Str("query", query).Any("param", params).Msg("SQL")
	}

	rows, err := p.Conn.Query(query, params...)
	if err != nil {
		p.logError(err)
		return items
	}
	defer rows.Close()

	for rows.Next() {
		var item Legaldoc
		err := rows.Scan(&item.Id, &item.Project, &item.Type, &item.Lang, &item.Version, &item.Title, &item.Body, &item.Effectivedate, &item.Date)
		if err != nil {
			p.logError(err)
			break
		}
		// DATE 컬럼이 드라이버 설정에 따라 시각까지 붙어 올 수 있다.
		if len(item.Effectivedate) > 10 {
			item.Effectivedate = item.Effectivedate[:10]
		}
		items = append(items, item)
	}

	return items
}

func (p *LegaldocManager) exec(query string, params ...interface{}) (sql.Result, error) {
	if p.Conn == nil || !p.Conn.IsConnect() {
		return nil, errors.New("Connection Error")
	}
	if p.Log {
		log.Debug().Str("query", query).Any("param", params).Msg("SQL")
	}
	res, err := p.Conn.Exec(query, params...)
	p.logError(err)
	return res, err
}

// Get 은 시행 여부와 상관없이 id 로 한 행을 읽는다.
func (p *LegaldocManager) Get(id int64) *Legaldoc {
	items := p.query("select "+legaldocColumns+" from legaldoc_tb where ld_id = ?", id)
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

// FindByProject 는 한 프로젝트의 모든 버전을 돌려준다(관리 화면용, 시행 전 포함).
func (p *LegaldocManager) FindByProject(project int64) []Legaldoc {
	return p.query("select "+legaldocColumns+" from legaldoc_tb where ld_project = ? order by ld_type, ld_lang, ld_version desc", project)
}

// FindEffective 는 today 까지 시행된 버전을 최신순으로 돌려준다. 첫 번째가 현재 문서다.
func (p *LegaldocManager) FindEffective(project int64, doctype string, lang string, today string) []Legaldoc {
	return p.query("select "+legaldocColumns+" from legaldoc_tb where ld_project = ? and ld_type = ? and ld_lang = ? and ld_effectivedate <= ? order by ld_version desc",
		project, doctype, lang, today)
}

// Insert 는 (프로젝트, 종류, 언어)의 다음 버전 번호를 매겨 새 행을 넣는다.
// 번호 계산과 삽입을 한 문장으로 해서, 동시에 두 번 등록돼도 유니크 키가 한쪽을 막는다.
func (p *LegaldocManager) Insert(item *Legaldoc) error {
	item.Date = time.Now().UTC().Add(9 * time.Hour).Format("2006-01-02 15:04:05")

	res, err := p.exec("insert into legaldoc_tb (ld_project, ld_type, ld_lang, ld_version, ld_title, ld_body, ld_effectivedate, ld_date) "+
		"select ?, ?, ?, coalesce(max(ld_version), 0) + 1, ?, ?, ?, ? from legaldoc_tb where ld_project = ? and ld_type = ? and ld_lang = ?",
		item.Project, item.Type, item.Lang, item.Title, item.Body, item.Effectivedate, item.Date,
		item.Project, item.Type, item.Lang)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	item.Id = id

	if saved := p.Get(id); saved != nil {
		item.Version = saved.Version
	}
	return nil
}

// UpdatePending 은 아직 시행 전인 행의 제목·본문·시행일만 고친다. 이미 시행된
// 행은 조건에 걸리지 않아 바뀌지 않는다 — 바뀐 행 수로 호출부가 구분한다.
func (p *LegaldocManager) UpdatePending(item *Legaldoc, today string) (int64, error) {
	res, err := p.exec("update legaldoc_tb set ld_title = ?, ld_body = ?, ld_effectivedate = ? where ld_id = ? and ld_effectivedate > ?",
		item.Title, item.Body, item.Effectivedate, item.Id, today)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeletePending 은 아직 시행 전인 행만 지운다.
func (p *LegaldocManager) DeletePending(id int64, today string) (int64, error) {
	res, err := p.exec("delete from legaldoc_tb where ld_id = ? and ld_effectivedate > ?", id, today)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
