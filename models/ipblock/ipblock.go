package ipblock

type Column int

const (
    _ Column = iota
    
    ColumnId
    ColumnAddress
    ColumnReason
    ColumnDate
)

type Params struct {
    Column Column
    Value interface{}
}




