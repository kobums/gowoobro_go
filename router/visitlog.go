package router

import (
	"net/netip"
	"strings"
	"sync"
	"time"

	"gowoobro/global/geoip"
	"gowoobro/global/log"
	"gowoobro/global/ntfy"
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
// 지오 조회(최대 5초)도 그 고루틴 안에서 한다. 봇은 IP 가 다양해 무료 한도만
// 축내므로 지오 조회를 생략한다.
var recordVisit = func(item models.Iplog, bot bool) {
	go func() {
		if !bot {
			g := geoip.Lookup(item.Address)
			item.Country, item.Region, item.City = g.Country, g.Region, g.City
			item.Isp, item.Org, item.As = g.ISP, g.Org, g.AS
		}

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

// notifyVisit 는 ntfy.sh 로 방문 알림을 보낸다(발송 자체는 global/ntfy).
// 테스트에서 실제 발송 없이 검증할 수 있도록 변수로 뒀다. 지오 조회는
// recordVisit 쪽과 중복될 수 있지만 캐시가 받아줘서 사실상 한 번이다.
var notifyVisit = func(item models.Iplog) {
	go func() {
		g := geoip.Lookup(item.Address)
		ntfy.Send("gowoobro 방문", visitDetail(item, g), []string{"eyes"})
	}()
}

// visitDetail 알림 본문: IP → 위치·통신사 → 기기 → 경로 순으로 아는 만큼만
func visitDetail(item models.Iplog, g geoip.Info) string {
	lines := []string{item.Address}
	if geo := joinParts(" · ", g.Country, joinParts(" ", g.Region, g.City), g.ISP); geo != "" {
		lines = append(lines, geo)
	}
	if dev := joinParts(" · ", item.Os, item.Browser); dev != "" {
		lines = append(lines, dev)
	}
	lines = append(lines, item.Path)
	return strings.Join(lines, "\n")
}

// joinParts 빈 문자열을 건너뛰고 구분자로 잇는다
func joinParts(sep string, parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

// isInternalIP 는 사설 대역(RFC 1918)·루프백 주소인지 판정한다. gowoobro_web
// 컨테이너의 SSR fetch 가 도커 네트워크 IP(172.22.x.x)로 들어오는데, 이건 방문이
// 아니라 우리 서버끼리의 통신이므로 방문 로그에서 걸러내기 위해 쓴다.
func isInternalIP(address string) bool {
	addr, err := netip.ParseAddr(address)
	if err != nil {
		return false
	}
	return addr.IsPrivate() || addr.IsLoopback()
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

	// 내부 트래픽(SSR fetch, 로컬 개발)은 적재도 알림도 하지 않는다. 실제 방문은
	// 방문자 브라우저가 직접 보내는 API 호출로 이미 잡힌다.
	if isInternalIP(address) {
		return c.Next()
	}

	agent := c.Get("User-Agent")
	os, browser, bot := parseAgent(agent)
	item := models.Iplog{Address: address, Path: c.Path(), Agent: agent, Os: os, Browser: browser}

	recordVisit(item, bot)

	// 봇 크롤링까지 알리면 알림이 폭주하므로 사람 방문만 알린다.
	if !bot && notified.shouldNotify(address, time.Now()) {
		notifyVisit(item)
	}

	return c.Next()
}
