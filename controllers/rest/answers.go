package rest

import (
	"gowoobro/controllers"
	"gowoobro/models"
	"strings"
)

type AnswersController struct {
	controllers.Controller
}

func (c *AnswersController) Read(id int64) {
	conn := c.NewConnection()

	manager := models.NewAnswersManager(conn)
	item := manager.Get(id)

	c.Set("item", item)
}

func (c *AnswersController) Index(page int, pagesize int) {
	conn := c.NewConnection()

	manager := models.NewAnswersManager(conn)

	var args []interface{}

	_address := c.Get("address")
	if _address != "" {
		args = append(args, models.Where{Column: "address", Value: _address, Compare: "="})
	}
	_question := c.Get("question")
	if _question != "" {
		args = append(args, models.Where{Column: "question", Value: _question, Compare: "like"})
	}
	_startdate := c.Get("startdate")
	_enddate := c.Get("enddate")
	if _startdate != "" && _enddate != "" {
		var v [2]string
		v[0] = _startdate
		v[1] = _enddate
		args = append(args, models.Where{Column: "date", Value: v, Compare: "between"})
	} else if _startdate != "" {
		args = append(args, models.Where{Column: "date", Value: _startdate, Compare: ">="})
	} else if _enddate != "" {
		args = append(args, models.Where{Column: "date", Value: _enddate, Compare: "<="})
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
					str += ", a_" + strings.Trim(v, " ")
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

func (c *AnswersController) Count() {
	conn := c.NewConnection()

	manager := models.NewAnswersManager(conn)

	var args []interface{}

	_address := c.Get("address")
	if _address != "" {
		args = append(args, models.Where{Column: "address", Value: _address, Compare: "="})
	}
	_question := c.Get("question")
	if _question != "" {
		args = append(args, models.Where{Column: "question", Value: _question, Compare: "like"})
	}
	_startdate := c.Get("startdate")
	_enddate := c.Get("enddate")

	if _startdate != "" && _enddate != "" {
		var v [2]string
		v[0] = _startdate
		v[1] = _enddate
		args = append(args, models.Where{Column: "date", Value: v, Compare: "between"})
	} else if _startdate != "" {
		args = append(args, models.Where{Column: "date", Value: _startdate, Compare: ">="})
	} else if _enddate != "" {
		args = append(args, models.Where{Column: "date", Value: _enddate, Compare: "<="})
	}

	total := manager.Count(args)
	c.Set("total", total)
}

func (c *AnswersController) Insert(item *models.Answers) {
	conn := c.NewConnection()

	if item.Address == "" {
		item.Address = c.Context.IP()
	}

	manager := models.NewAnswersManager(conn)
	err := manager.Insert(item)
	if err != nil {
		c.Set("code", "error")
		c.Set("error", err)
		return
	}

	id := manager.GetIdentity()
	c.Result["id"] = id
	item.Id = id
}

func (c *AnswersController) Insertbatch(item *[]models.Answers) {
	if item == nil || len(*item) == 0 {
		return
	}

	rows := len(*item)

	conn := c.NewConnection()

	manager := models.NewAnswersManager(conn)

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

func (c *AnswersController) Update(item *models.Answers) {
	conn := c.NewConnection()

	manager := models.NewAnswersManager(conn)
	err := manager.Update(item)
	if err != nil {
		c.Set("code", "error")
		c.Set("error", err)
		return
	}
}

func (c *AnswersController) Delete(item *models.Answers) {
	conn := c.NewConnection()

	manager := models.NewAnswersManager(conn)

	err := manager.Delete(item.Id)
	if err != nil {
		c.Set("code", "error")
		c.Set("error", err)
	}
}

func (c *AnswersController) Deletebatch(item *[]models.Answers) {
	conn := c.NewConnection()

	manager := models.NewAnswersManager(conn)

	for _, v := range *item {
		err := manager.Delete(v.Id)
		if err != nil {
			c.Set("code", "error")
			c.Set("error", err)
			return
		}
	}
}
