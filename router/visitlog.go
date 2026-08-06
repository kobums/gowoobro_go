package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"gowoobro/global/config"
	"gowoobro/global/log"
	"gowoobro/models"

	"github.com/gofiber/fiber/v2"
	"github.com/mileusna/useragent"
)

// parseAgent 는 User-Agent 문자열에서 OS 와 브라우저를 뽑는다. 통계 조회 때
// LIKE 검색 대신 바로 GROUP BY 할 수 있도록 적재 시점에 파싱해둔다.
//
// 봇(Googlebot 등)은 브라우저가 아니므로 browser 자리에 봇 이름을 넣는다 —
// il_browser 만 봐도 사람/봇이 구분되게 하기 위해서다.
func parseAgent(agent string) (string, string, bool) {
	if agent == "" {
		return "", "", false
	}

	ua := useragent.Parse(agent)

	os := strings.TrimSpace(ua.OS + " " + ua.OSVersion)

	// 전체 버전(126.0.6478.127)은 통계를 잘게 쪼개기만 하므로 메이저 버전까지만.
	version := ua.Version
	if idx := strings.Index(version, "."); idx > 0 {
		version = version[:idx]
	}
	browser := strings.TrimSpace(ua.Name + " " + version)

	return os, browser, ua.Bot
}

// 알림은 "요청"이 아니라 "방문" 단위로 보낸다. 페이지 하나를 열어도 API 가 여러 번
// 불리므로, 같은 IP 는 notifyTTL 안에서 한 번만 알린다. DB 적재는 요청마다 전부
// 되고, 이 묶음은 알림에만 적용된다.
const notifyTTL = 5 * time.Minute

// notifySeenLimit 를 넘으면 만료된 항목을 청소한다. 크롤러가 IP 를 바꿔가며
// 훑어도 메모리가 무한히 크지 않게 막아둔다.
const notifySeenLimit = 10000

type notifyCache struct {
	mutex sync.Mutex
	seen  map[string]time.Time
}

var notified = notifyCache{seen: map[string]time.Time{}}

func (n *notifyCache) shouldNotify(address string, now time.Time) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if last, ok := n.seen[address]; ok && now.Sub(last) < notifyTTL {
		return false
	}

	if len(n.seen) >= notifySeenLimit {
		for key, last := range n.seen {
			if now.Sub(last) >= notifyTTL {
				delete(n.seen, key)
			}
		}
	}

	n.seen[address] = now
	return true
}

// recordVisit 는 iplog_tb 에 한 건 적재한다. 테스트에서 DB 없이 호출을 검증할 수
// 있도록 변수로 뒀다. 기본 구현은 요청 처리를 막지 않게 고루틴에서 돈다.
var recordVisit = func(item models.Iplog) {
	go func() {
		conn := models.NewConnection()
		if conn == nil || !conn.IsConnect() {
			log.Error().Str("address", item.Address).Msg("visitlog: database connection error")
			return
		}
		defer conn.Close()

		manager := models.NewIplogManager(conn)
		if err := manager.Insert(&item); err != nil {
			// 방문 로그는 부가 기능이다. 실패해도 요청에는 영향을 주지 않고
			// 로그만 남긴다.
			log.Error().Str("error", err.Error()).Str("address", item.Address).Msg("visitlog: insert failed")
		}
	}()
}

var ntfyClient = &http.Client{Timeout: 10 * time.Second}

// notifyVisit 는 ntfy.sh 로 방문 알림을 보낸다. topic 이름이 곧 인증이므로
// (아는 사람은 누구나 구독 가능) NTFY_TOPIC 환경변수로만 받고, 없으면 아무것도
// 하지 않는다. 테스트에서 실제 발송 없이 검증할 수 있도록 변수로 뒀다.
var notifyVisit = func(item models.Iplog) {
	topic := config.NtfyTopic
	if topic == "" {
		return
	}

	go func() {
		detail := item.Address
		if item.Os != "" || item.Browser != "" {
			detail += " · " + strings.TrimSpace(strings.Trim(item.Os+" · "+item.Browser, " ·"))
		}
		detail += "\n" + item.Path

		// 제목에 한글을 쓰려면 HTTP 헤더 대신 JSON 발행을 써야 한다
		// (헤더는 ASCII 만 안전하다).
		payload, err := json.Marshal(map[string]interface{}{
			"topic":   topic,
			"title":   "gowoobro 방문",
			"message": detail,
			"tags":    []string{"eyes"},
		})
		if err != nil {
			return
		}

		resp, err := ntfyClient.Post("https://ntfy.sh", "application/json", bytes.NewReader(payload))
		if err != nil {
			log.Error().Str("error", err.Error()).Msg("visitlog: ntfy send failed")
			return
		}
		resp.Body.Close()
	}()
}

// VisitLog 는 요청마다 방문자 IP 를 iplog_tb 에 적재하고, 사람 방문이면 ntfy 로
// 푸시 알림을 보낸다. 응답을 기다리게 하지 않도록 둘 다 비동기로 한다.
func VisitLog(c *fiber.Ctx) error {
	// CORS preflight 는 브라우저 내부 동작이라 방문으로 치지 않는다.
	if c.Method() == fiber.MethodOptions {
		return c.Next()
	}

	address := clientIP(c)
	if address == "" {
		return c.Next()
	}

	agent := c.Get("User-Agent")
	os, browser, bot := parseAgent(agent)
	item := models.Iplog{Address: address, Path: c.Path(), Agent: agent, Os: os, Browser: browser}

	recordVisit(item)

	// 봇 크롤링까지 알리면 알림이 폭주하므로 사람 방문만 알린다.
	if !bot && notified.shouldNotify(address, time.Now()) {
		notifyVisit(item)
	}

	return c.Next()
}
