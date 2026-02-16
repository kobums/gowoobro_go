package ipblock

type Column int

const (
    _ Column = iota
    
    ColumnId
    ColumnAddress
    ColumnDate
)

type Params struct {
    Column Column
    Value interface{}
}




