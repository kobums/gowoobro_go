-- 방문 기록 테이블. ipblock_tb 가 차단 목록 전용이 된 뒤 방문 IP 적재를 이어받는다.
-- 앱은 INSERT 만 하고, 조회는 phpMyAdmin 등 DB 에서 직접 한다.
--
-- il_path  : 어떤 API 를 조회했는지. 취약점 스캐너 판별(차단 등록 근거)용.
-- il_agent : User-Agent. 봇/크롤러와 실제 방문 구분용.
CREATE TABLE IF NOT EXISTS iplog_tb (
    il_id      BIGINT       NOT NULL AUTO_INCREMENT COMMENT '방문 기록 ID',
    il_address VARCHAR(45)  NOT NULL COMMENT 'IP 주소',
    il_path    VARCHAR(255) NOT NULL DEFAULT '' COMMENT '요청 경로',
    il_agent   VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'User-Agent',
    il_date    DATETIME     NOT NULL COMMENT '기록일',
    PRIMARY KEY (il_id),
    KEY idx_iplog_address (il_address),
    KEY idx_iplog_date (il_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 운영 DB(go_mariadb)에는 il_id, il_address, il_date 3컬럼짜리 테이블이 이미
-- 있으므로 위 CREATE 는 건너뛰어지고, 아래 ALTER 로 두 컬럼을 추가해야 한다.
-- (MariaDB 10.0+ 는 ADD COLUMN IF NOT EXISTS 를 지원한다)
ALTER TABLE iplog_tb
    ADD COLUMN IF NOT EXISTS il_path  VARCHAR(255) NOT NULL DEFAULT '' COMMENT '요청 경로' AFTER il_address,
    ADD COLUMN IF NOT EXISTS il_agent VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'User-Agent' AFTER il_path;
