package rest

import (
	"gowoobro/controllers"
	"gowoobro/global/ntfy"
	"gowoobro/models"

	"strings"
)

type QuestionsController struct {
	controllers.Controller
}

func (c *QuestionsController) Read(id int64) {
    
    
	conn := c.NewConnection()

	manager := models.NewQuestionsManager(conn)
	item := manager.Get(id)

    
    
    c.Set("item", item)
}

func (c *QuestionsController) Index(page int, pagesize int) {
    
    
	conn := c.NewConnection()

	manager := models.NewQuestionsManager(conn)

    var args []interface{}
    
    _address := c.Get("address")
    if _address != "" {
        args = append(args, models.Where{Column:"address", Value:_address, Compare:"like"})
    }
    _question := c.Get("question")
    if _question != "" {
        args = append(args, models.Where{Column:"question", Value:_question, Compare:"like"})
    }
    _startdate := c.Get("startdate")
    _enddate := c.Get("enddate")
    if _startdate != "" && _enddate != "" {        
        var v [2]string
        v[0] = _startdate
        v[1] = _enddate  
        args = append(args, models.Where{Column:"date", Value:v, Compare:"between"})    
    } else if  _startdate != "" {          
        args = append(args, models.Where{Column:"date", Value:_startdate, Compare:">="})
    } else if  _enddate != "" {          
        args = append(args, models.Where{Column:"date", Value:_enddate, Compare:"<="})            
    }
    

    
    
    if page != 0 && pagesize != 0 {
        args = append(args, models.Paging(page, pagesize))
    }
    
    orderby := c.Get("orderby")
    if orderby == "" {
        if page != 0 && pagesize != 0 {
            orderby = "id desc"
            args = append(args, models.Ordering(orderby))
        }
    } else {
        orderbys := strings.Split(orderby, ",")

        str := ""
        for i, v := range orderbys {
            if i == 0 {
                str += v
            } else {
                if strings.Contains(v, "_") {                   
                    str += ", " + strings.Trim(v, " ")
                } else {
                    str += ", q_" + strings.Trim(v, " ")                
                }
            }
        }
        
        args = append(args, models.Ordering(str))
    }
    
	items := manager.Find(args)
	c.Set("items", items)

    if page == 1 {
       total := manager.Count(args)
	   c.Set("total", total)
    }
}

func (c *QuestionsController) Count() {
    
    
	conn := c.NewConnection()

	manager := models.NewQuestionsManager(conn)

    var args []interface{}
    
    _address := c.Get("address")
    if _address != "" {
        args = append(args, models.Where{Column:"address", Value:_address, Compare:"like"})
        
    }
    _question := c.Get("question")
    if _question != "" {
        args = append(args, models.Where{Column:"question", Value:_question, Compare:"like"})
        
    }
    _startdate := c.Get("startdate")
    _enddate := c.Get("enddate")

    if _startdate != "" && _enddate != "" {        
        var v [2]string
        v[0] = _startdate
        v[1] = _enddate  
        args = append(args, models.Where{Column:"date", Value:v, Compare:"between"})    
    } else if  _startdate != "" {          
        args = append(args, models.Where{Column:"date", Value:_startdate, Compare:">="})
    } else if  _enddate != "" {          
        args = append(args, models.Where{Column:"date", Value:_enddate, Compare:"<="})            
    }
    
    
    
    
    total := manager.Count(args)
	c.Set("total", total)
}

func (c *QuestionsController) Insert(item *models.Questions) {
    
    
	conn := c.NewConnection()
    
    // 답변 조회가 서버가 본 IP 로 걸러지므로, 질문도 같은 IP 로 적재해야 짝이 맞는다.
    if visitor := c.VisitorAddress(); visitor != "" {
        item.Address = visitor
    }

    if item.Address == "" {
        item.Address = c.Context.IP()
    }
    
	manager := models.NewQuestionsManager(conn)
	err := manager.Insert(item)
    if err != nil {
        c.Set("code", "error")    
        c.Set("error", err)
        return
    }

    id := manager.GetIdentity()
    c.Result["id"] = id
    item.Id = id

    notifyQuestion(*item)
}

// notifyQuestion 은 방문자 질문이 접수되면 방문 알림과 같은 ntfy 채널로 알린다.
// 발송이 접수 응답을 막으면 안 되므로 고루틴으로 보낸다.
var notifyQuestion = func(item models.Questions) {
	go func() {
		question := item.Question
		// 알림은 미리보기 용도라 너무 길면 자른다. 전문은 관리자 페이지에서 본다.
		if runes := []rune(question); len(runes) > 200 {
			question = string(runes[:200]) + "…"
		}
		ntfy.Send("gowoobro 새 질문", question+"\n— "+item.Address, []string{"speech_balloon"})
	}()
}

func (c *QuestionsController) Insertbatch(item *[]models.Questions) {  
    if item == nil || len(*item) == 0 {
        return
    }

    rows := len(*item)
    
    
    
	conn := c.NewConnection()
    
	manager := models.NewQuestionsManager(conn)

    for i := 0; i < rows; i++ {
        if ((*item)[i]).Address == "" {
            ((*item)[i]).Address = c.Context.IP()
        }
	    err := manager.Insert(&((*item)[i]))
        if err != nil {
            c.Set("code", "error")    
            c.Set("error", err)
            return
        }
    }
}

func (c *QuestionsController) Update(item *models.Questions) {
    
    
	conn := c.NewConnection()

	manager := models.NewQuestionsManager(conn)
    err := manager.Update(item)
    if err != nil {
        c.Set("code", "error")    
        c.Set("error", err)
        return
    }
}

func (c *QuestionsController) Delete(item *models.Questions) {
    
    
    conn := c.NewConnection()

	manager := models.NewQuestionsManager(conn)

    
	err := manager.Delete(item.Id)
    if err != nil {
        c.Set("code", "error")    
        c.Set("error", err)
    }
}

func (c *QuestionsController) Deletebatch(item *[]models.Questions) {
    
    
    conn := c.NewConnection()

	manager := models.NewQuestionsManager(conn)

    for _, v := range *item {
        
    
	    err := manager.Delete(v.Id)
        if err != nil {
            c.Set("code", "error")    
            c.Set("error", err)
            return
        }
    }
}


