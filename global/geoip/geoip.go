package geoip

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// IP 지오로케이션 — ip-api.com 무료 API (비상업, 분당 45회 제한)로 국가·지역·
// 도시·통신사를 조회한다 (ninedragons_back 의 server/geoip.go 와 같은 방식).
// 방문 로그 보강용이라 실패해도 빈 값으로 넘어간다. IP당 한 번만 조회해
// 캐시하고, 무료 한도 아래로 자체 제한한다. HTTPS 는 유료 플랜이라 HTTP 를
// 쓴다 — 공개 IP 의 위치 메타데이터라 수용한다. 한국어 로케일은 ip-api 가
// 지원하지 않아 영문 그대로 받는다.

type Info struct {
	Country string `json:"country"`
	Region  string `json:"regionName"`
	City    string `json:"city"`
	ISP     string `json:"isp"`
	Org     string `json:"org"`
	AS      string `json:"as"`
}

const (
	cacheLimit  = 10000
	ratePerMin  = 40 // ip-api 무료 한도(45/분) 아래로
	lookupFlags = "status,country,regionName,city,isp,org,as"
)

var (
	mu     sync.Mutex
	cache  = map[string]Info{}
	window time.Time
	count  int
	client = &http.Client{Timeout: 5 * time.Second}
)

// allow 분당 조회 수 제한 (mu 잠금은 호출자 책임)
func allow(now time.Time) bool {
	if now.Sub(window) >= time.Minute {
		window = now
		count = 0
	}
	if count >= ratePerMin {
		return false
	}
	count++
	return true
}

// Lookup 은 IP 의 위치·통신사 정보를 조회한다. 캐시 우선이고, 실패·한도 초과
// 시 빈 값을 주되 캐시하지 않아 다음 방문 때 재시도된다. 최대 5초 블로킹하므로
// 요청 처리 경로가 아닌 고루틴에서 불러야 한다. 테스트에서 실제 호출 없이
// 검증할 수 있도록 변수로 뒀다.
var Lookup = func(ip string) Info {
	mu.Lock()
	if g, ok := cache[ip]; ok {
		mu.Unlock()
		return g
	}
	allowed := allow(time.Now())
	mu.Unlock()
	if !allowed {
		return Info{}
	}

	resp, err := client.Get("http://ip-api.com/json/" + ip + "?fields=" + lookupFlags)
	if err != nil {
		return Info{}
	}
	defer resp.Body.Close()

	var body struct {
		Status string `json:"status"`
		Info
	}
	if json.NewDecoder(resp.Body).Decode(&body) != nil || body.Status != "success" {
		return Info{}
	}

	mu.Lock()
	if len(cache) >= cacheLimit {
		// 개인 규모에선 도달할 일이 드물다 — 단순 전체 비움으로 충분
		cache = map[string]Info{}
	}
	cache[ip] = body.Info
	mu.Unlock()
	return body.Info
}
