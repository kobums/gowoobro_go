package rest

import (
	"gowoobro/controllers"
	"gowoobro/models"

	"strings"
)

// IpblockController 는 차단 IP 목록(ipblock_tb)의 조회만 담당한다.
//
// 차단 IP 는 DB 에 직접 INSERT 로 넣는다. 인증 없는 쓰기 엔드포인트를 열어두면
// 아무나 차단 목록을 고칠 수 있어서 Insert/Update/Delete 계열은 전부 제거했다.
//
// 주의: 이 파일은 gomachine 생성기 산출물이다. 생성기를 다시 돌리면 쓰기
// 메서드가 되살아나므로 다시 지워야 한다.
type IpblockController struct {
	controllers.Controller
}

func (c *IpblockController) Read(id int64) {
    
    
	conn := c.NewConnection()

	manager := models.NewIpblockManager(conn)
	item := manager.Get(id)

    
    
    c.Set("item", item)
}

func (c *IpblockController) Index(page int, pagesize int) {
    
    
	conn := c.NewConnection()

	manager := models.NewIpblockManager(conn)

    var args []interface{}
    
    _address := c.Get("address")
    if _address != "" {
        args = append(args, models.Where{Column:"address", Value:_address, Compare:"like"})
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
                    str += ", ib_" + strings.Trim(v, " ")                
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

func (c *IpblockController) Count() {
    
    
	conn := c.NewConnection()

	manager := models.NewIpblockManager(conn)

    var args []interface{}
    
    _address := c.Get("address")
    if _address != "" {
        args = append(args, models.Where{Column:"address", Value:_address, Compare:"like"})
        
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
