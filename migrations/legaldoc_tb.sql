-- 약관·개인정보 처리방침 등 법적 문서. 앱/웹을 새로 낼 때마다 페이지를 만들어
-- 배포하지 않도록, 여기에 행을 넣으면 gowoobro_web 이 페이지로 그려준다.
--   https://gowoobro.com/{lang}/app/{프로젝트 id}/{ld_type}
--
-- 고치지 않고 쌓는다. 내용을 바꿀 때는 ld_version 을 올린 새 행을 넣는다.
-- 시행일(ld_effectivedate)이 지난 것 중 가장 높은 버전이 현재 문서이고,
-- 이전 버전도 계속 열람할 수 있다. 시행 전인 행만 수정·삭제할 수 있다.
--
-- ld_project       : projects_tb.p_id
-- ld_type          : URL 에 쓰이는 문서 종류. terms, privacy, location 등 (소문자·숫자·하이픈)
-- ld_lang          : ko, en
-- ld_version       : (프로젝트, 종류, 언어) 안에서 1부터 서버가 매긴다
-- ld_title         : 페이지 제목. "개인정보 처리방침" 등
-- ld_body          : 본문 (마크다운)
-- ld_effectivedate : 시행일. 이 날부터 공개된다
CREATE TABLE IF NOT EXISTS legaldoc_tb (
    ld_id            BIGINT       NOT NULL AUTO_INCREMENT COMMENT '문서 ID',
    ld_project       BIGINT       NOT NULL COMMENT '프로젝트 ID',
    ld_type          VARCHAR(30)  NOT NULL COMMENT '문서 종류',
    ld_lang          VARCHAR(5)   NOT NULL DEFAULT 'ko' COMMENT '언어',
    ld_version       INT          NOT NULL COMMENT '버전',
    ld_title         VARCHAR(100) NOT NULL COMMENT '제목',
    ld_body          MEDIUMTEXT   NOT NULL COMMENT '본문(마크다운)',
    ld_effectivedate DATE         NOT NULL COMMENT '시행일',
    ld_date          DATETIME     NOT NULL COMMENT '등록일',
    PRIMARY KEY (ld_id),
    UNIQUE KEY uk_legaldoc_version (ld_project, ld_type, ld_lang, ld_version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
