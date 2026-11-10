package rest

import (
	"errors"
	"net/http"

	"gowoobro/controllers"
	"gowoobro/models"
)

// LegaldocController 는 약관·개인정보 처리방침 등 법적 문서를 다룬다.
//
// 공개 조회(Current, Read)는 시행일이 지난 버전만 내준다. 시행 전 초안은
// 관리자 전용인 Index 로만 볼 수 있다. 어느 경로가 공개인지는 router/auth.go 의
// isPublicPath 가 정한다.
type LegaldocController struct {
	controllers.Controller
}

// legaldocVersion 은 버전 목록에 쓰는 요약이다. 본문은 뺀다.
type legaldocVersion struct {
	Id            int64  `json:"id"`
	Version       int    `json:"version"`
	Effectivedate string `json:"effectivedate"`
}

func (c *LegaldocController) fail(code int, err error) {
	c.Code = code
	c.Error(err)
}

func (c *LegaldocController) setVersions(items []models.Legaldoc) {
	versions := make([]legaldocVersion, 0, len(items))
	for _, v := range items {
		versions = append(versions, legaldocVersion{Id: v.Id, Version: v.Version, Effectivedate: v.Effectivedate})
	}
	c.Set("versions", versions)
}

// Index 는 한 프로젝트의 모든 버전을 돌려준다(관리자 전용, 시행 전 포함).
func (c *LegaldocController) Index(project int64) {
	if project <= 0 {
		c.fail(http.StatusBadRequest, errors.New("project is required"))
		return
	}

	manager := models.NewLegaldocManager(c.NewConnection())
	c.Set("items", manager.FindByProject(project))
	c.Set("today", models.LegaldocToday())
}

// Current 는 지금 시행 중인 문서와 그동안의 버전 목록을 돌려준다.
// 요청한 언어의 문서가 없으면 한국어 문서로 대신한다.
func (c *LegaldocController) Current(project int64, doctype string, lang string) {
	manager := models.NewLegaldocManager(c.NewConnection())
	today := models.LegaldocToday()

	if lang == "" {
		lang = "ko"
	}

	items := manager.FindEffective(project, doctype, lang, today)
	if len(items) == 0 && lang != "ko" {
		items = manager.FindEffective(project, doctype, "ko", today)
	}

	if len(items) == 0 {
		c.Set("item", nil)
		return
	}

	c.Set("item", items[0])
	c.setVersions(items)
}

// Read 는 특정 버전을 돌려준다. 시행 전인 버전은 없는 것으로 취급한다.
func (c *LegaldocController) Read(id int64) {
	manager := models.NewLegaldocManager(c.NewConnection())
	today := models.LegaldocToday()

	item := manager.Get(id)
	if item == nil || item.Effectivedate > today {
		c.Set("item", nil)
		return
	}

	c.Set("item", item)
	c.setVersions(manager.FindEffective(item.Project, item.Type, item.Lang, today))
}

// Insert 는 새 버전을 등록한다. 버전 번호는 서버가 매긴다.
func (c *LegaldocController) Insert(item *models.Legaldoc) {
	item.Normalize()
	if err := item.Validate(); err != nil {
		c.fail(http.StatusBadRequest, err)
		return
	}

	conn := c.NewConnection()

	if models.NewProjectsManager(conn).Get(item.Project) == nil {
		c.fail(http.StatusBadRequest, errors.New("project not found"))
		return
	}

	manager := models.NewLegaldocManager(conn)
	if err := manager.Insert(item); err != nil {
		c.fail(http.StatusInternalServerError, errors.New("insert failed"))
		return
	}

	c.Set("id", item.Id)
	c.Set("version", item.Version)
}

// pending 은 id 의 문서가 아직 시행 전이라 고칠 수 있는지 확인한다.
func (c *LegaldocController) pending(manager *models.LegaldocManager, id int64, today string) *models.Legaldoc {
	saved := manager.Get(id)
	if saved == nil {
		c.fail(http.StatusNotFound, errors.New("not found"))
		return nil
	}
	if saved.Effectivedate <= today {
		// 이미 공개된 문서는 사용자가 동의한 내용이다. 고치지 않고 새 버전을 쌓는다.
		c.fail(http.StatusConflict, errors.New("already effective; register a new version instead"))
		return nil
	}
	return saved
}

// Update 는 시행 전인 버전의 제목·본문·시행일을 고친다.
func (c *LegaldocController) Update(item *models.Legaldoc) {
	manager := models.NewLegaldocManager(c.NewConnection())
	today := models.LegaldocToday()

	saved := c.pending(manager, item.Id, today)
	if saved == nil {
		return
	}

	// 어느 프로젝트의 어떤 문서인지는 바꿀 수 없다. 버전 번호가 거기에 묶여 있다.
	item.Project, item.Type, item.Lang = saved.Project, saved.Type, saved.Lang
	item.Normalize()
	if err := item.Validate(); err != nil {
		c.fail(http.StatusBadRequest, err)
		return
	}

	if _, err := manager.UpdatePending(item, today); err != nil {
		c.fail(http.StatusInternalServerError, errors.New("update failed"))
	}
}

// Delete 는 시행 전인 버전을 지운다.
func (c *LegaldocController) Delete(id int64) {
	manager := models.NewLegaldocManager(c.NewConnection())
	today := models.LegaldocToday()

	if c.pending(manager, id, today) == nil {
		return
	}

	if _, err := manager.DeletePending(id, today); err != nil {
		c.fail(http.StatusInternalServerError, errors.New("delete failed"))
	}
}
