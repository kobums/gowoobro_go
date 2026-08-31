package ntfy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"gowoobro/global/config"
	"gowoobro/global/log"
)

var client = &http.Client{Timeout: 10 * time.Second}

// Send 는 ntfy.sh 로 푸시 알림을 보낸다. topic 이름이 곧 인증이므로(아는 사람은
// 누구나 구독 가능) NTFY_TOPIC 환경변수로만 받고, 없으면 아무것도 하지 않는다.
//
// 동기로 보내므로 요청 처리 경로에서는 고루틴으로 감싸서 호출해야 한다.
// 제목·본문에 한글을 쓰려면 HTTP 헤더 대신 JSON 발행을 써야 한다
// (헤더는 ASCII 만 안전하다).
func Send(title, message string, tags []string) {
	topic := config.NtfyTopic
	if topic == "" {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"topic":   topic,
		"title":   title,
		"message": message,
		"tags":    tags,
	})
	if err != nil {
		return
	}

	resp, err := client.Post("https://ntfy.sh", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Error().Str("error", err.Error()).Str("title", title).Msg("ntfy: send failed")
		return
	}
	resp.Body.Close()
}
