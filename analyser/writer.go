package analyser

import (
	"regexp"
	"sort"

	"github.com/boreevyuri/clitable"
)

// TableWriter составитель таблиц
type TableWriter interface {

	// добавляет идентификатора по ключу
	Add(string, int)

	// экспортирует данные от одного составителя другому
	Export(TableWriter)

	// возвращает идентификатору по ключу
	Ids() map[string][]int

	// устанавливает регулярное выражение для ключей
	SetKeyPattern(string)

	// устанавливает лимит
	SetLimit(int)

	// сигнализирует, нужен ли список email-ов после таблицы
	SetNecessaryExport(bool)

	// устанавливает сдвиг
	SetOffset(int)

	// устанавливает строки для вывода в таблице
	SetRows(RowWriters)

	// устанавливает регулярное выражение для значения строки таблицы
	SetValuePattern(string)

	// выводит ьаблицу
	Show()
}

// RowWriters строки таблицы
type RowWriters map[int]RowWriter

// RowWriter строка таблицы
type RowWriter interface {

	// записывает строку в таблицу, если строка удовлетворяет регулярному выражению
	Write(*clitable.Table, *regexp.Regexp)
}

// AbstractTableWriter базовый автор таблицы
type AbstractTableWriter struct {
	*clitable.Table
	ids             map[string][]int
	keyPattern      string
	limit           int
	necessaryExport bool
	offset          int
	rows            RowWriters
	valuePattern    string
}

// создает базовый рисователь таблицы
func newAbstractTableWriter(fields []interface{}) *AbstractTableWriter {
	return &AbstractTableWriter{
		Table: clitable.NewTable(fields...),
		ids:   make(map[string][]int),
	}
}

// Add добавляет идентификатора по ключу
func (a *AbstractTableWriter) Add(key string, id int) {
	if _, ok := a.ids[key]; !ok {
		a.ids[key] = make([]int, 0)
	}
	idsLen := len(a.ids[key])
	if sort.Search(idsLen, func(i int) bool { return a.ids[key][i] >= id }) == idsLen {
		a.ids[key] = append(a.ids[key], id)
	}
}

// Export экспортирует данные от одного составителя другому
func (a *AbstractTableWriter) Export(writer TableWriter) {
	a.ids = writer.Ids()
}

// Ids возвращает идентификатору по ключу
func (a *AbstractTableWriter) Ids() map[string][]int {
	return a.ids
}

// SetKeyPattern устанавливает регулярное выражение для ключей
func (a *AbstractTableWriter) SetKeyPattern(pattern string) {
	a.keyPattern = pattern
}

// SetLimit устанавливает лимит
func (a *AbstractTableWriter) SetLimit(limit int) {
	a.limit = limit
}

// SetNecessaryExport сигнализирует, нужен ли список email-ов после таблицы
func (a *AbstractTableWriter) SetNecessaryExport(necessaryExport bool) {
	a.necessaryExport = necessaryExport
}

// SetOffset устанавливает сдвиг
func (a *AbstractTableWriter) SetOffset(offset int) {
	a.offset = offset
}

// SetRows устанавливает строки для вывода в таблице
func (a *AbstractTableWriter) SetRows(rows RowWriters) {
	a.rows = rows
}

// SetValuePattern устанавливает регулярное выражение для значения строки таблицы
func (a *AbstractTableWriter) SetValuePattern(pattern string) {
	a.valuePattern = pattern
}
