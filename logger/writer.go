package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/boreevyuri/postmanq/common"
)

// Writer писатель логов.
type Writer interface {
	writeString(string)
}

// StdoutWriter писатель логов пишущий в стандартный вывод.
type StdoutWriter struct{}

// writeString пишет логи в стандартный вывод.
func (s *StdoutWriter) writeString(str string) {
	_, _ = os.Stdout.WriteString(str)
}

// FileWriter писатель логов в файл.
type FileWriter struct {
	filename string
}

// writeString пишет логи в файл.
func (fw *FileWriter) writeString(str string) {
	file, err := os.OpenFile(fw.filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err == nil {
		_, _ = file.WriteString(str)

		// TODO: обработать ошибку Close
		file.Close()
	}
}

// Writers массив писателей логов.
type Writers []Writer

// количество писателей.
func (w Writers) len() int {
	return len(w)
}

// init инициализирует писателей логов.
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

// set добавляет писателя в список.
func (w *Writers) set(i int, writer Writer) {
	(*w)[i] = writer
}

// run запускает писателей.
func (w Writers) run() {
	for _, writer := range w {
		go w.listenMessages(writer)
	}
}

// listenMessages подписывает писателей на получение сообщений для логирования.
func (w *Writers) listenMessages(writer Writer) {
	for message := range messages {
		w.writeMessage(writer, message, time.Now().Format(time.StampMicro))
	}
}

// writeMessage пишет сообщение в лог
func (w *Writers) writeMessage(writer Writer, message *Message, date string) {
	writer.writeString(
		fmt.Sprintf(
			"PostmanQ | %v | %s: %s\n",
			// time.Now().Format("2006-01-02 15:04:05"),
			// time.Now().Format(time.StampMicro),
			date,
			logLevelByID[message.Level],
			fmt.Sprintf(message.Message, message.Args...),
		),
	)
}
