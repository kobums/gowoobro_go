package rest


import (
	"gowoobro/controllers"
	"gowoobro/models"

    "strings"
)

type ProjectsController struct {
	controllers.Controller
}

func (c *ProjectsController) Read(id int64) {
    
    
	conn := c.NewConnection()

	manager := models.NewProjectsManager(conn)
	item := manager.Get(id)

    
    
    c.Set("item", item)
}

func (c *ProjectsController) Index(page int, pagesize int) {
    
    
	conn := c.NewConnection()

	manager := models.NewProjectsManager(conn)

    var args []interface{}
    
    _key := c.Get("key")
    if _key != "" {
        args = append(args, models.Where{Column:"key", Value:_key, Compare:"like"})
    }
    _type := c.Get("type")
    if _type != "" {
        args = append(args, models.Where{Column:"type", Value:_type, Compare:"like"})
    }
    _title := c.Get("title")
    if _title != "" {
        args = append(args, models.Where{Column:"title", Value:_title, Compare:"="})
        
    }
    _description := c.Get("description")
    if _description != "" {
        args = append(args, models.Where{Column:"description", Value:_description, Compare:"like"})
    }
    _iconurl := c.Get("iconurl")
    if _iconurl != "" {
        args = append(args, models.Where{Column:"iconurl", Value:_iconurl, Compare:"like"})
    }
    _url := c.Get("url")
    if _url != "" {
        args = append(args, models.Where{Column:"url", Value:_url, Compare:"like"})
    }
    _playstoreurl := c.Get("playstoreurl")
    if _playstoreurl != "" {
        args = append(args, models.Where{Column:"playstoreurl", Value:_playstoreurl, Compare:"like"})
    }
    _appstoreurl := c.Get("appstoreurl")
    if _appstoreurl != "" {
        args = append(args, models.Where{Column:"appstoreurl", Value:_appstoreurl, Compare:"like"})
    }
    _qrcodeurl := c.Get("qrcodeurl")
    if _qrcodeurl != "" {
        args = append(args, models.Where{Column:"qrcodeurl", Value:_qrcodeurl, Compare:"like"})
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
                    str += ", p_" + strings.Trim(v, " ")                
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

func (c *ProjectsController) Count() {
    
    
	conn := c.NewConnection()

	manager := models.NewProjectsManager(conn)

    var args []interface{}
    
    _key := c.Get("key")
    if _key != "" {
        args = append(args, models.Where{Column:"key", Value:_key, Compare:"like"})
        
    }
    _type := c.Get("type")
    if _type != "" {
        args = append(args, models.Where{Column:"type", Value:_type, Compare:"like"})
        
    }
    _title := c.Get("title")
    if _title != "" {
        args = append(args, models.Where{Column:"title", Value:_title, Compare:"="})
        
        
    }
    _description := c.Get("description")
    if _description != "" {
        args = append(args, models.Where{Column:"description", Value:_description, Compare:"like"})
        
    }
    _iconurl := c.Get("iconurl")
    if _iconurl != "" {
        args = append(args, models.Where{Column:"iconurl", Value:_iconurl, Compare:"like"})
        
    }
    _url := c.Get("url")
    if _url != "" {
        args = append(args, models.Where{Column:"url", Value:_url, Compare:"like"})
        
    }
    _playstoreurl := c.Get("playstoreurl")
    if _playstoreurl != "" {
        args = append(args, models.Where{Column:"playstoreurl", Value:_playstoreurl, Compare:"like"})
        
    }
    _appstoreurl := c.Get("appstoreurl")
    if _appstoreurl != "" {
        args = append(args, models.Where{Column:"appstoreurl", Value:_appstoreurl, Compare:"like"})
        
    }
    _qrcodeurl := c.Get("qrcodeurl")
    if _qrcodeurl != "" {
        args = append(args, models.Where{Column:"qrcodeurl", Value:_qrcodeurl, Compare:"like"})
        
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

func (c *ProjectsController) Insert(item *models.Projects) {
    
    
	conn := c.NewConnection()
    
	manager := models.NewProjectsManager(conn)
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

func (c *ProjectsController) Insertbatch(item *[]models.Projects) {  
    if item == nil || len(*item) == 0 {
        return
    }

    rows := len(*item)
    
    
    
	conn := c.NewConnection()
    
	manager := models.NewProjectsManager(conn)

    for i := 0; i < rows; i++ {
	    err := manager.Insert(&((*item)[i]))
        if err != nil {
            c.Set("code", "error")    
            c.Set("error", err)
            return
        }
    }
}

func (c *ProjectsController) Update(item *models.Projects) {
    
    
	conn := c.NewConnection()

	manager := models.NewProjectsManager(conn)
    err := manager.Update(item)
    if err != nil {
        c.Set("code", "error")    
        c.Set("error", err)
        return
    }
}

func (c *ProjectsController) Delete(item *models.Projects) {
    
    
    conn := c.NewConnection()

	manager := models.NewProjectsManager(conn)

    
	err := manager.Delete(item.Id)
    if err != nil {
        c.Set("code", "error")    
        c.Set("error", err)
    }
}

func (c *ProjectsController) Deletebatch(item *[]models.Projects) {
    
    
    conn := c.NewConnection()

	manager := models.NewProjectsManager(conn)

    for _, v := range *item {
        
    
	    err := manager.Delete(v.Id)
        if err != nil {
            c.Set("code", "error")    
            c.Set("error", err)
            return
        }
    }
}


