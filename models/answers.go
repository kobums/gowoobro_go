package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gowoobro/global/config"
	log "gowoobro/global/log"
	"gowoobro/models/answers"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type Answers struct {
	Id           int64                  `json:"id"`
	Address      string                 `json:"address"`
	Question     int64                  `json:"question"`
	QuestionText string                 `json:"questionText"`
	Answer       string                 `json:"answer"`
	Date         string                 `json:"date"`
	Extra        map[string]interface{} `json:"extra"`
}

type AnswersManager struct {
	Conn        *Connection
	Result      *sql.Result
	Index       string
	Isolation   bool
	SelectQuery string
	JoinQuery   string
	CountQuery  string
	GroupQuery  string
	SelectLog   bool
	Log         bool
}

func (c *Answers) AddExtra(key string, value interface{}) {
	c.Extra[key] = value
}

func NewAnswersManager(conn *Connection) *AnswersManager {
	var item AnswersManager

	if conn == nil {
		item.Conn = NewConnection()
		item.Isolation = false
	} else {
		item.Conn = conn
		item.Isolation = conn.Isolation
	}

	item.Index = ""
	item.SelectLog = config.Log.Database
	item.Log = config.Log.Database

	return &item
}

func (p *AnswersManager) Close() {
	if p.Conn != nil {
		p.Conn.Close()
	}
}

func (p *AnswersManager) SetIndex(index string) {
	p.Index = index
}

func (p *AnswersManager) SetCountQuery(query string) {
	p.CountQuery = query
}

func (p *AnswersManager) SetSelectQuery(query string) {
	p.SelectQuery = query
}

func (p *AnswersManager) Exec(query string, params ...interface{}) (sql.Result, error) {
	if p.Log {
		if len(params) > 0 {
			log.Debug().Str("query", query).Any("param", params).Msg("SQL")
		} else {
			log.Debug().Str("query", query).Msg("SQL")
		}
	}
	return p.Conn.Exec(query, params...)
}

func (p *AnswersManager) Query(query string, params ...interface{}) (*sql.Rows, error) {
	if p.Isolation {
		query += " for update"
	}

	if p.SelectLog {
		if len(params) > 0 {
			log.Debug().Str("query", query).Any("param", params).Msg("SQL")
		} else {
			log.Debug().Str("query", query).Msg("SQL")
		}
	}

	return p.Conn.Query(query, params...)
}

func (p *AnswersManager) GetQuery() string {
	if p.SelectQuery != "" {
		return p.SelectQuery
	}

	var ret strings.Builder
	ret.WriteString("select a_id, a_address, a_question, a_answer, a_date, coalesce(q_question, '') from answers_tb left join questions_tb on a_question = q_id")

	if p.Index != "" {
		ret.WriteString(" use index(")
		ret.WriteString(p.Index)
		ret.WriteString(")")
	}

	if p.JoinQuery != "" {
		ret.WriteString(", ")
		ret.WriteString(p.JoinQuery)
	}

	ret.WriteString(" where 1=1 ")

	return ret.String()
}

func (p *AnswersManager) GetQuerySelect() string {
	if p.CountQuery != "" {
		return p.CountQuery
	}

	var ret strings.Builder
	ret.WriteString("select count(*) from answers_tb")

	if p.Index != "" {
		ret.WriteString(" use index(")
		ret.WriteString(p.Index)
		ret.WriteString(")")
	}

	if p.JoinQuery != "" {
		ret.WriteString(", ")
		ret.WriteString(p.JoinQuery)
	}

	ret.WriteString(" where 1=1 ")

	return ret.String()
}

func (p *AnswersManager) GetQueryGroup(name string) string {
	if p.SelectQuery != "" {
		return p.SelectQuery
	}

	var ret strings.Builder
	ret.WriteString("select a_")
	ret.WriteString(name)
	ret.WriteString(", count(*) from answers_tb ")

	if p.Index != "" {
		ret.WriteString(" use index(")
		ret.WriteString(p.Index)
		ret.WriteString(")")
	}

	ret.WriteString(" where 1=1 ")

	return ret.String()
}

func (p *AnswersManager) Truncate() error {
	if !p.Conn.IsConnect() {
		return errors.New("Connection Error")
	}

	query := "truncate answers_tb "
	_, err := p.Exec(query)

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
	}

	return nil
}

func (p *AnswersManager) Insert(item *Answers) error {
	if !p.Conn.IsConnect() {
		return errors.New("Connection Error")
	}

	if item.Date == "" {
		t := time.Now().UTC().Add(time.Hour * 9)
		item.Date = fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	}

	if item.Date == "" {
		item.Date = "1000-01-01 00:00:00"
	}

	query := ""
	var res sql.Result
	var err error
	if item.Id > 0 {
		query = "insert into answers_tb (a_id, a_address, a_question, a_answer, a_date) values (?, ?, ?, ?, ?)"
		res, err = p.Exec(query, item.Id, item.Address, item.Question, item.Answer, item.Date)
	} else {
		query = "insert into answers_tb (a_address, a_question, a_answer, a_date) values (?, ?, ?, ?)"
		res, err = p.Exec(query, item.Address, item.Question, item.Answer, item.Date)
	}
	// item.Question is the FK (q_id); QuestionText is read-only from JOIN

	if err == nil {
		p.Result = &res
	} else {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
		p.Result = nil
	}

	return err
}

func (p *AnswersManager) Delete(id int64) error {
	if !p.Conn.IsConnect() {
		return errors.New("Connection Error")
	}

	query := "delete from answers_tb where a_id = ?"
	_, err := p.Exec(query, id)

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
	}

	return err
}

func (p *AnswersManager) DeleteAll() error {
	if !p.Conn.IsConnect() {
		return errors.New("Connection Error")
	}

	query := "delete from answers_tb"
	_, err := p.Exec(query)

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
	}

	return err
}

func (p *AnswersManager) MakeQuery(initQuery string, postQuery string, initParams []interface{}, args []interface{}) (string, []interface{}) {
	var params []interface{}
	if initParams != nil {
		params = append(params, initParams...)
	}

	pos := 1

	var query strings.Builder
	query.WriteString(initQuery)

	for _, arg := range args {
		switch v := arg.(type) {
		case Where:
			item := v

			if strings.Contains(item.Column, "_") {
				query.WriteString(" and ")
			} else {
				query.WriteString(" and a_")
			}
			query.WriteString(item.Column)

			if item.Compare == "in" {
				query.WriteString(" in (")
				query.WriteString(strings.Trim(strings.Replace(fmt.Sprint(item.Value), " ", ", ", -1), "[]"))
				query.WriteString(")")
			} else if item.Compare == "not in" {
				query.WriteString(" not in (")
				query.WriteString(strings.Trim(strings.Replace(fmt.Sprint(item.Value), " ", ", ", -1), "[]"))
				query.WriteString(")")
			} else if item.Compare == "between" {
				if config.Database.Type == config.Postgresql {
					query.WriteString(fmt.Sprintf(" between $%v and $%v", pos, pos+1))
					pos += 2
				} else {
					query.WriteString(" between ? and ?")
				}

				s := item.Value.([2]string)
				params = append(params, s[0])
				params = append(params, s[1])
			} else {
				if config.Database.Type == config.Postgresql {
					query.WriteString(" ")
					query.WriteString(item.Compare)
					query.WriteString(fmt.Sprintf(" $%v", pos))
					pos++
				} else {
					query.WriteString(" ")
					query.WriteString(item.Compare)
					query.WriteString(" ?")
				}
				if item.Compare == "like" {
					params = append(params, "%"+item.Value.(string)+"%")
				} else {
					params = append(params, item.Value)
				}
			}
		case Custom:
			item := v

			query.WriteString(" and ")
			query.WriteString(item.Query)
		}
	}

	query.WriteString(postQuery)

	return query.String(), params
}

func (p *AnswersManager) DeleteWhere(args []interface{}) error {
	if !p.Conn.IsConnect() {
		return errors.New("Connection Error")
	}

	query, params := p.MakeQuery("delete from answers_tb where 1=1", "", nil, args)
	_, err := p.Exec(query, params...)

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
	}

	return err
}

func (p *AnswersManager) Update(item *Answers) error {
	if !p.Conn.IsConnect() {
		return errors.New("Connection Error")
	}

	if item.Date == "" {
		item.Date = "1000-01-01 00:00:00"
	}

	query := "update answers_tb set a_address = ?, a_question = ?, a_answer = ?, a_date = ? where a_id = ?"
	_, err := p.Exec(query, item.Address, item.Question, item.Answer, item.Date, item.Id)
	// item.Question is the FK (q_id); QuestionText is derived from JOIN

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
	}

	return err
}

func (p *AnswersManager) UpdateWhere(columns []answers.Params, args []interface{}) error {
	if !p.Conn.IsConnect() {
		return errors.New("Connection Error")
	}

	var initQuery strings.Builder
	var initParams []interface{}

	initQuery.WriteString("update answers_tb set ")
	for i, v := range columns {
		if i > 0 {
			initQuery.WriteString(", ")
		}

		if v.Column == answers.ColumnId {
			initQuery.WriteString("a_id = ?")
			initParams = append(initParams, v.Value)
		} else if v.Column == answers.ColumnAddress {
			initQuery.WriteString("a_address = ?")
			initParams = append(initParams, v.Value)
		} else if v.Column == answers.ColumnQuestion {
			initQuery.WriteString("a_question = ?")
			initParams = append(initParams, v.Value)
		} else if v.Column == answers.ColumnAnswer {
			initQuery.WriteString("a_answer = ?")
			initParams = append(initParams, v.Value)
		} else if v.Column == answers.ColumnDate {
			initQuery.WriteString("a_date = ?")
			initParams = append(initParams, v.Value)
		}
	}

	initQuery.WriteString(" where 1=1 ")

	query, params := p.MakeQuery(initQuery.String(), "", initParams, args)
	_, err := p.Exec(query, params...)

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
	}

	return err
}

func (p *AnswersManager) GetIdentity() int64 {
	if !p.Conn.IsConnect() {
		return 0
	}

	id, err := (*p.Result).LastInsertId()

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
		return 0
	} else {
		return id
	}
}

func (p *Answers) InitExtra() {
	p.Extra = map[string]interface{}{}
}

func (p *AnswersManager) ReadRow(rows *sql.Rows) *Answers {
	var item Answers
	var err error

	if rows.Next() {
		err = rows.Scan(&item.Id, &item.Address, &item.Question, &item.Answer, &item.Date, &item.QuestionText)

		if item.Date == "0000-00-00 00:00:00" || item.Date == "1000-01-01 00:00:00" || item.Date == "9999-01-01 00:00:00" {
			item.Date = ""
		}

		if config.Database.Type == config.Postgresql {
			item.Date = strings.ReplaceAll(strings.ReplaceAll(item.Date, "T", " "), "Z", "")
		}
	} else {
		return nil
	}

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
		return nil
	} else {
		item.InitExtra()
		return &item
	}
}

func (p *AnswersManager) ReadRows(rows *sql.Rows) []Answers {
	var items []Answers

	for rows.Next() {
		var item Answers

		err := rows.Scan(&item.Id, &item.Address, &item.Question, &item.Answer, &item.Date, &item.QuestionText)
		if err != nil {
			if p.Log {
				log.Error().Str("error", err.Error()).Msg("SQL")
			}
			break
		}

		if item.Date == "0000-00-00 00:00:00" || item.Date == "1000-01-01 00:00:00" || item.Date == "9999-01-01 00:00:00" {
			item.Date = ""
		}

		if config.Database.Type == config.Postgresql {
			item.Date = strings.ReplaceAll(strings.ReplaceAll(item.Date, "T", " "), "Z", "")
		}

		item.InitExtra()
		items = append(items, item)
	}

	return items
}

func (p *AnswersManager) Get(id int64) *Answers {
	if !p.Conn.IsConnect() {
		return nil
	}

	var query strings.Builder
	query.WriteString(p.GetQuery())
	query.WriteString(" and a_id = ?")

	rows, err := p.Query(query.String(), id)

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
		return nil
	}

	defer rows.Close()

	return p.ReadRow(rows)
}

func (p *AnswersManager) GetWhere(args []interface{}) *Answers {
	items := p.Find(args)
	if len(items) == 0 {
		return nil
	}

	return &items[0]
}

func (p *AnswersManager) Count(args []interface{}) int {
	if !p.Conn.IsConnect() {
		return 0
	}

	query, params := p.MakeQuery(p.GetQuerySelect(), p.GroupQuery, nil, args)
	rows, err := p.Query(query, params...)

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
		return 0
	}

	defer rows.Close()

	if !rows.Next() {
		return 0
	}

	cnt := 0
	err = rows.Scan(&cnt)

	if err != nil {
		return 0
	} else {
		return cnt
	}
}

func (p *AnswersManager) FindAll() []Answers {
	return p.Find(nil)
}

func (p *AnswersManager) Find(args []interface{}) []Answers {
	if !p.Conn.IsConnect() {
		var items []Answers
		return items
	}

	var params []interface{}
	baseQuery := p.GetQuery()

	var query strings.Builder

	page := 0
	pagesize := 0
	orderby := ""

	pos := 1

	for _, arg := range args {
		switch v := arg.(type) {
		case PagingType:
			item := v
			page = item.Page
			pagesize = item.Pagesize
		case OrderingType:
			item := v
			orderby = item.Order
		case LimitType:
			item := v
			page = 1
			pagesize = item.Limit
		case OptionType:
			item := v
			if item.Limit > 0 {
				page = 1
				pagesize = item.Limit
			} else {
				page = item.Page
				pagesize = item.Pagesize
			}
			orderby = item.Order
		case Where:
			item := v

			if strings.Contains(item.Column, "_") {
				query.WriteString(" and ")
			} else {
				query.WriteString(" and a_")
			}
			query.WriteString(item.Column)

			if item.Compare == "in" {
				query.WriteString(" in (")
				query.WriteString(strings.Trim(strings.Replace(fmt.Sprint(item.Value), " ", ", ", -1), "[]"))
				query.WriteString(")")
			} else if item.Compare == "not in" {
				query.WriteString(" not in (")
				query.WriteString(strings.Trim(strings.Replace(fmt.Sprint(item.Value), " ", ", ", -1), "[]"))
				query.WriteString(")")
			} else if item.Compare == "between" {
				if config.Database.Type == config.Postgresql {
					query.WriteString(fmt.Sprintf(" between $%v and $%v", pos, pos+1))
					pos += 2
				} else {
					query.WriteString(" between ? and ?")
				}

				s := item.Value.([2]string)
				params = append(params, s[0])
				params = append(params, s[1])
			} else {
				if config.Database.Type == config.Postgresql {
					query.WriteString(" ")
					query.WriteString(item.Compare)
					query.WriteString(fmt.Sprintf(" $%v", pos))
					pos++
				} else {
					query.WriteString(" ")
					query.WriteString(item.Compare)
					query.WriteString(" ?")
				}
				if item.Compare == "like" {
					params = append(params, "%"+item.Value.(string)+"%")
				} else {
					params = append(params, item.Value)
				}
			}
		case Custom:
			item := v

			query.WriteString(" and ")
			query.WriteString(item.Query)
		case Base:
			item := v

			baseQuery = item.Query
		}
	}

	query.WriteString(p.GroupQuery)

	startpage := (page - 1) * pagesize

	if page > 0 && pagesize > 0 {
		if orderby == "" {
			orderby = "a_id desc"
		} else {
			if !strings.Contains(orderby, "_") {
				if strings.ToUpper(orderby) != "RAND()" {
					orderby = "a_" + orderby
				}
			}
		}
		query.WriteString(" order by ")
		query.WriteString(orderby)
		if config.Database.Type == config.Postgresql {
			query.WriteString(fmt.Sprintf(" limit $%v offset $%v", pos, pos+1))
			params = append(params, pagesize)
			params = append(params, startpage)
		} else if config.Database.Type == config.Mysql {
			query.WriteString(" limit ? offset ?")
			params = append(params, pagesize)
			params = append(params, startpage)
		} else if config.Database.Type == config.Sqlserver {
			query.WriteString("OFFSET ? ROWS FETCH NEXT ? ROWS ONLY")
			params = append(params, startpage)
			params = append(params, pagesize)
		}
	} else {
		if orderby == "" {
			orderby = "a_id desc"
		} else {
			if !strings.Contains(orderby, "_") {
				if strings.ToUpper(orderby) != "RAND()" {
					orderby = "a_" + orderby
				}
			}
		}
		query.WriteString(" order by ")
		query.WriteString(orderby)
	}

	rows, err := p.Query(baseQuery+query.String(), params...)

	if err != nil {
		if p.Log {
			log.Error().Str("error", err.Error()).Msg("SQL")
		}
		items := make([]Answers, 0)
		return items
	}

	defer rows.Close()

	return p.ReadRows(rows)
}
