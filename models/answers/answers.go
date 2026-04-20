package answers

type Column int

const (
	_ Column = iota

	ColumnId
	ColumnAddress
	ColumnQuestion
	ColumnAnswer
	ColumnDate
)

type Params struct {
	Column Column
	Value  interface{}
}
