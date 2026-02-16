package questions

type Column int

const (
    _ Column = iota
    
    ColumnId
    ColumnAddress
    ColumnQuestion
    ColumnDate
)

type Params struct {
    Column Column
    Value interface{}
}




