package common

import (
	"flag"
	"fmt"
	"regexp"
	"runtime"
)

const (
	// ExampleConfigYaml используется в примерах использования
	ExampleConfigYaml = "/path/to/config/file.yaml"

	// InvalidInputString невалидная строка, введенная пользователем
	InvalidInputString = ""

	// InvalidInputInt невалидное число, введенное пользователем
	InvalidInputInt = 0
)

var (
	// App объект текущего приложения, иногда необходим сервисам, для отправки событий приложению
	App Application

	// Services сервисы, используются для создания итератора
	Services []interface{}

	// DefaultWorkersCount количество goroutine, может измениться для инициализации приложения
	DefaultWorkersCount = runtime.NumCPU()

	// FilenameRegex используется в нескольких пакетах, поэтому вынес сюда
	FilenameRegex = regexp.MustCompile(`[^\\/]+\.[^\\/]+`)

	// PrintUsage печает аргументы, используемые приложением
	PrintUsage = func(f *flag.Flag) {
		format := "  -%s %s\n"
		fmt.Printf(format, f.Name, f.Usage)
	}
)

// Application проект содержит несколько приложений: pmq-grep, pmq-publish, pmq-report, postmanq и т.д.
// чтобы упростить и стандартизировать приложения, разработан этот интерфейс.
type Application interface {
	// устанавливает путь к файлу с настройками
	SetConfigFilename(string)

	// проверяет валидность пути к файлу с настройками
	IsValidConfigFilename(string) bool

	// устанавливает канал событий приложения
	SetEvents(chan *ApplicationEvent)

	// возвращает канал событий приложения
	Events() chan *ApplicationEvent

	// устанавливает канал завершения приложения
	SetDone(chan bool)

	// возвращает канал завершения приложения
	Done() chan bool

	// возвращает сервисы, используемые приложением
	Services() []interface{}

	// инициализирует сервисы
	FireInit(*ApplicationEvent, interface{})

	// запускает сервисы приложения
	FireRun(*ApplicationEvent, interface{})

	// останавливает сервисы приложения
	FireFinish(*ApplicationEvent, interface{})

	// инициализирует приложение
	Init(*ApplicationEvent)

	// запускает приложение
	Run()

	// запускает приложение с аргументами
	RunWithArgs(...interface{})

	// возвращает таймауты приложения
	Timeout() Timeout
}
