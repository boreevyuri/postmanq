package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/boreevyuri/postmanq/common"
)

// Writer автор логов
type Writer interface {
	writeString(string)
}

// StdoutWriter автор логов пишущий в стандартный вывод
type StdoutWriter struct{}

// writeString пишет логи в стандартный вывод
func (this *StdoutWriter) writeString(str string) {
	os.Stdout.WriteString(str)
}

// FileWriter автор логов пишущий в файл
type FileWriter struct {
	filename string
}

// writeString пишет логи в файл
func (this *FileWriter) writeString(str string) {
	f, err := os.OpenFile(this.filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err == nil {
		_, err = f.WriteString(str)
		f.Close()
	}
}

// Writers писатели логов
type Writers []Writer

// количество писателей
func (w Writers) len() int {
	return len(w)
}

// init инициализирует писателей логов
func (w *Writers) init() {
	for i := 0; i < w.len(); i++ {
		if common.FilenameRegex.MatchString(service.Output) { // проверяем получили ли из настроек имя файла
			// получаем директорию, в которой лежит файл
			dir := filepath.Dir(service.Output)
			// смотрим, что она реально существует
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				FailExit("directory %s is not exists", dir)
			} else {
				w.set(i, &FileWriter{service.Output})
			}
		} else if len(service.Output) == 0 || service.Output == "stdout" {
			w.set(i, &StdoutWriter{})
		}
	}
}

// set добавляет писателя в список
func (w *Writers) set(i int, writer Writer) {
	(*w)[i] = writer
}

// write запускает писателей
func (w Writers) write() {
	for _, writer := range w {
		go w.listenMessages(writer)
	}
}

// listenMessages подписывает писателей на получение сообщений для логирования
func (w *Writers) listenMessages(writer Writer) {
	for message := range messages {
		w.writeMessage(writer, message)
	}
}

// writeMessage пишет сообщение в лог
func (w *Writers) writeMessage(writer Writer, message *Message) {
	writer.writeString(
		fmt.Sprintf(
			"PostmanQ | %v | %s: %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			logLevelByID[message.Level],
			fmt.Sprintf(message.Message, message.Args...),
		),
	)
}
