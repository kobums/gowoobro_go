package router

import (
	"errors"
	"strings"
	"sync"
	"time"

	"gowoobro/global/log"
	"gowoobro/global/setting"
	"gowoobro/models"

	"github.com/gofiber/fiber/v2"
)

// ipblock_tb 는 "접근을 차단할 IP 목록"이다. 예전에는 방문한 사용자의 IP 를
// 그대로 쌓아두는 기록용이었지만 지금은 정반대 — 이 테이블에 들어있는 IP 는
// /api 전체에서 막힌다.
//
// ib_address 는 setting.NewIP 가 파싱하므로 단일 주소("1.2.3.4") 외에
// 대역("1.2.3.0/24")과 범위("1.2.3.10-20") 표기도 그대로 쓸 수 있다.
//
// 차단 IP 는 애플리케이션이 아니라 DB 에 직접 INSERT 로 넣는다(쓰기 API 없음).
// 그래서 앱은 목록이 언제 바뀌는지 알 방법이 없고, blocklistTTL 주기로 다시 읽는다.
// 즉 DB 에 넣은 차단이 실제로 걸리기까지 최대 blocklistTTL 만큼 걸린다.
const blocklistTTL = 60 * time.Second

// blockRule 은 차단 규칙 한 줄이다. Reason 은 막힌 사람에게 그대로 보여주므로
// 방문자가 읽을 문장으로 적어야 한다.
type blockRule struct {
	IP     *setting.IP
	Reason string
}

type blocklistCache struct {
	mutex    sync.RWMutex
	entries  []blockRule
	loadedAt time.Time
	loaded   bool
}

var blocklist blocklistCache

// Entries 는 캐시가 유효하면 그대로, 만료됐으면 DB 에서 다시 읽어 반환한다.
func (c *blocklistCache) Entries() []blockRule {
	c.mutex.RLock()
	if c.loaded && time.Since(c.loadedAt) < blocklistTTL {
		cached := c.entries
		c.mutex.RUnlock()
		return cached
	}
	c.mutex.RUnlock()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 읽기 잠금을 놓고 쓰기 잠금을 잡는 사이에 다른 요청이 이미 갱신했을 수 있다.
	if c.loaded && time.Since(c.loadedAt) < blocklistTTL {
		return c.entries
	}

	entries, err := loadBlocklist()
	if err != nil {
		// DB 가 죽었다고 사이트를 통째로 열거나 막지 않는다. 직전 캐시를 그대로
		// 유지하고 다음 요청에서 다시 시도한다.
		log.Error().Str("error", err.Error()).Msg("ipblock: reload failed, keeping previous cache")
		return c.entries
	}

	c.entries = entries
	c.loadedAt = time.Now()
	c.loaded = true

	return c.entries
}

func loadBlocklist() ([]blockRule, error) {
	conn := models.NewConnection()
	if conn == nil || !conn.IsConnect() {
		return nil, errors.New("database connection error")
	}
	defer conn.Close()

	// sql.Open 은 실제로 접속하지 않는다. 여기서 Ping 으로 확인해두지 않으면 DB 가
	// 죽었을 때 FindAll 이 빈 슬라이스를 돌려주고, 그게 "차단 목록이 비어있다"와
	// 구분되지 않아 캐시가 조용히 비워진다.
	if err := conn.Conn.Ping(); err != nil {
		return nil, err
	}

	manager := models.NewIpblockManager(conn)

	var entries []blockRule
	for _, item := range manager.FindAll() {
		address := strings.TrimSpace(item.Address)
		if address == "" {
			continue
		}

		entry, err := setting.NewIP(address)
		if err != nil {
			// setting.NewIP 는 IPv4 전용이다. 파싱 못 하는 행은 건너뛴다.
			log.Warn().Int64("id", item.Id).Str("address", address).Msg("ipblock: skip unparsable rule")
			continue
		}

		entries = append(entries, blockRule{IP: entry, Reason: strings.TrimSpace(item.Reason)})
	}

	return entries, nil
}

// clientIP 는 요청을 보낸 실제 사용자의 IP 를 구한다.
//
// fiber.Config 에 ProxyHeader 가 설정돼 있지 않아서 c.IP() 는 리버스 프록시
// 컨테이너의 IP 를 돌려준다. 전역 설정을 켜면 프록시가 없는 로컬 개발에서 빈 값이
// 되어 questions/answers 컨트롤러의 기존 폴백까지 망가지므로, 여기서만 직접 뽑는다.
func clientIP(c *fiber.Ctx) string {
	// nginx-proxy 는 X-Real-IP 를 $remote_addr 로 "덮어쓴다". 클라이언트가 같은
	// 헤더를 실어 보내도 프록시가 지워버리므로 위조가 안 된다. 그래서 최우선.
	if address := strings.TrimSpace(c.Get("X-Real-IP")); address != "" {
		return address
	}

	// X-Forwarded-For 는 $proxy_add_x_forwarded_for 로 클라이언트가 보낸 값 "뒤에"
	// 실제 IP 를 덧붙인다. 맨 앞 값은 위조될 수 있으니 X-Real-IP 가 없을 때만 쓴다.
	if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
		if address := strings.TrimSpace(strings.Split(forwarded, ",")[0]); address != "" {
			return address
		}
	}

	// 프록시를 거치지 않는 로컬 개발.
	return c.IP()
}

// IpBlockGuard 는 ipblock_tb 에 등록된 IP 의 요청을 403 으로 끊는다.
func IpBlockGuard(c *fiber.Ctx) error {
	address := clientIP(c)
	if address == "" {
		return c.Next()
	}

	ip, err := setting.NewIP(address)
	if err != nil {
		// 판정할 수 없는 주소(IPv6 등)는 통과시킨다.
		return c.Next()
	}

	for _, rule := range blocklist.Entries() {
		if !rule.IP.Contains(ip) {
			continue
		}

		log.Info().
			Str("address", address).
			Str("rule", rule.IP.Address).
			Str("reason", rule.Reason).
			Str("path", c.Path()).
			Msg("ipblock: access denied")

		// reason 은 막힌 사람에게 그대로 보여준다. 비어 있으면 프론트가 기존
		// 일반 오류 문구로 대체하도록 빈 문자열을 그대로 내려보낸다.
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"code":    "error",
			"message": "access denied",
			"reason":  rule.Reason,
		})
	}

	return c.Next()
}
