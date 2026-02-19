package projects

type Column int

const (
    _ Column = iota
    
    ColumnId
    ColumnKey
    ColumnType
    ColumnTitle
    ColumnDescription
    ColumnIconurl
    ColumnUrl
    ColumnPlaystoreurl
    ColumnAppstoreurl
    ColumnQrcodeurl
    ColumnDate
)

type Params struct {
    Column Column
    Value interface{}
}




