package analyser

import (
	"regexp"
	"time"

	"github.com/boreevyuri/clitable"
)

// Report отчет об ошибке
type Report struct {
	// идентификатор
	ID int

	// отправитель
	Envelope string

	// получатель
	Recipient string

	// код ошибки
	Code int

	// сообщение об ошибке
	Message string

	// даты отправок
	CreatedDates []time.Time
}

// записывает отчет в таблицу
func (r Report) Write(table *clitable.Table, valueRegex *regexp.Regexp) {
	if valueRegex == nil ||
		(valueRegex != nil &&
			(valueRegex.MatchString(r.Envelope) ||
				valueRegex.MatchString(r.Recipient) ||
				valueRegex.MatchString(r.Message))) {
		table.AddRow(
			r.Envelope,
			r.Recipient,
			r.Code,
			r.Message,
			len(r.CreatedDates),
		)
	}
}

// AggregateRow агрегированная строка
type AggregateRow []int

// записывает строку в таблицу
func (a AggregateRow) Write(table *clitable.Table, valueRegex *regexp.Regexp) {
	table.AddRow(a[0], a[1], a[2], a[3])
}
