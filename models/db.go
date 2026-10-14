package models

import (
	"fmt"
	"gowoobro/global/config"
	"gowoobro/global/log"

	"database/sql"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type PagingType struct {
	Page     int
	Pagesize int
}

type OrderingType struct {
	Order string
}

type LimitType struct {
	Limit int
}

type OptionType struct {
	Page     int
	Pagesize int
	Order    string
	Limit    int
}

type Where struct {
	Column  string
	Value   interface{}
	Compare string
}

type Custom struct {
	Query string
}

type Base struct {
	Query string
}

type Groupby struct {
	Value int `json:"value"`
	Count int `json:"count"`
}

func Paging(page int, pagesize int) PagingType {
	return PagingType{Page: page, Pagesize: pagesize}
}

// orderTerm 은 정렬 항목 하나("id", "p_date desc", "t.col asc")의 허용 형태다.
var orderTerm = regexp.MustCompile(`(?i)^[a-z_][a-z0-9_]*(\.[a-z_][a-z0-9_]*)?(\s+(asc|desc))?$`)

// sanitizeOrder 는 정렬 문자열에서 "컬럼 [asc|desc]" 나열과 rand() 만 통과시킨다.
//
// order by 절은 플레이스홀더로 바인딩할 수 없어 쿼리에 그대로 이어 붙는데, 이
// 값은 공개 API 의 orderby 쿼리스트링에서 온다. 걸러내지 않으면 서브쿼리를 넣어
// 다른 테이블 내용을 읽어낼 수 있다. 하나라도 형태가 어긋나면 통째로 버리고,
// 빈 값은 각 매니저의 기본 정렬로 떨어진다.
func sanitizeOrder(order string) string {
	var terms []string
	for _, term := range strings.Split(order, ",") {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		if !strings.EqualFold(term, "rand()") && !orderTerm.MatchString(term) {
			return ""
		}
		terms = append(terms, term)
	}

	return strings.Join(terms, ", ")
}

func Ordering(order string) OrderingType {
	return OrderingType{Order: sanitizeOrder(order)}
}

func Limit(limit int) LimitType {
	return LimitType{Limit: limit}
}

type Connection struct {
	Conn *sql.DB
    Tx    *sql.Tx    
	Transaction bool
	Isolation bool
}

func (c *Connection) Close() {
	c.Conn.Close()
}

func (c *Connection) IsConnect() bool {
	return c.Conn != nil
}

func (c *Connection) Exec(query string, params ...interface{}) (sql.Result, error) {
	if c.Transaction {
       return c.Tx.Exec(query, params...)    
	} else {
       return c.Conn.Exec(query, params...)
    }
}

func (c *Connection) Query(query string, params ...interface{}) (*sql.Rows, error) {
	if c.Transaction {
       return c.Tx.Query(query, params...)    
	} else {
       return c.Conn.Query(query, params...)
    }
}

func (c *Connection) Begin() {
	if c.Transaction {
		return
	}

	c.Tx, _ = c.Conn.Begin()
	c.Transaction = true
	c.Isolation = true
}

func (c *Connection) Commit() error {
	c.Transaction = false
	return c.Tx.Commit()
}

func (c *Connection) Rollback() {
	if !c.Transaction {
		return
	}

	err := c.Tx.Rollback()
	if err != nil {
		log.Error().Msg(err.Error())
	}
	c.Transaction = false
}

func GetConnection() *Connection {
	conn, err := sql.Open(config.Database.TypeString, config.Database.ConnectionString)
	if err != nil {
		log.Error().Msg(err.Error())
		return nil
	}

	conn.SetMaxOpenConns(100)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(5 * time.Minute)

	return &Connection{
		Conn: conn,
		Tx: nil,
		Transaction: false,
	}
}

func NewConnection() *Connection {
	db := GetConnection()

	if db != nil {
		return db
	}

	time.Sleep(100 * time.Millisecond)

	db = GetConnection()

	if db != nil {
		return db
	}

	time.Sleep(500 * time.Millisecond)

	db = GetConnection()

	if db != nil {
		return db
	}

	time.Sleep(1 * time.Second)

	db = GetConnection()

	if db != nil {
		return db
	}

	time.Sleep(2 * time.Second)

	db = GetConnection()

	return db
}

func QueryArray(db *Connection, query string, items []interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	rows, err = db.Conn.Query(query, items...)
	return rows, err
}

func ExecArray(db *Connection, query string, items []interface{}) error {
	var err error

	_, err = db.Conn.Exec(query, items...)
	return err
}

func InitDate() string {
	return "1000-01-01 00:00:00"
}

type Double float64

func (c Double) MarshalJSON() ([]byte, error) {
	if float64(c) == float64(int(c)) {
		return []byte(fmt.Sprintf("%v.0", int64(c))), nil
	}

	return []byte(fmt.Sprintf("%v", float64(c))), nil
}
