package logger

import (
	"github.com/boreevyuri/postmanq/common"
	"gopkg.in/yaml.v2"
)

// Level уровень логирования.
type Level int

// уровни логирования.
const (
	DebugLevel Level = iota
	InfoLevel
	WarningLevel
	ErrorLevel
)

// название уровней логирования.
const (
	DebugLevelName   = "debug"
	InfoLevelName    = "info"
	WarningLevelName = "warning"
	ErrorLevelName   = "error"
)

var (
	// названия уровней логирования, используется непосредственно в момент создания записи в лог
	logLevelByID = map[Level]string{
		DebugLevel:   DebugLevelName,
		InfoLevel:    InfoLevelName,
		WarningLevel: WarningLevelName,
		ErrorLevel:   ErrorLevelName,
	}
	// уровни логирования по названию, используется для удобной инициализации сервиса логирования
	logLevelByName = map[string]Level{
		DebugLevelName:   DebugLevel,
		InfoLevelName:    InfoLevel,
		WarningLevelName: WarningLevel,
		ErrorLevelName:   ErrorLevel,
	}
	messages = make(chan *Message)
	writers  = make(Writers, common.DefaultWorkersCount)
	level    = WarningLevel
	service  *Service
)

// Message запись логирования.
type Message struct {
	// сообщение для лога, может содержать параметры
	Message string

	// уровень логирования записи, необходим для отсечения лишних записей
	Level Level

	// аргументы для параметров сообщения
	Args []interface{}
}

// NewMessage создание новой записи логирования.
func NewMessage(level Level, message string, args ...interface{}) *Message {
	logMessage := new(Message)
	logMessage.Level = level
	logMessage.Message = message
	logMessage.Args = args

	return logMessage
}

// Service сервис логирования.
type Service struct {
	// название уровня логирования, устанавливается в конфиге
	LevelName string `yaml:"logLevel"`

	// название вывода логов
	Output string `yaml:"logOutput"`

	// уровень логов, ниже этого уровня логи писаться не будут
	level Level

	// куда пишем логи stdout или файл
	writer Writer

	// канал логирования
	messages chan *Message
}

// Inst создает новый сервис логирования.
func Inst() common.SendingService {
	if service == nil {
		service = new(Service)
		// запускаем запись логов в отдельном потоке
		writers.init()
		writers.run()
	}

	return service
}

// OnInit инициализирует сервис логирования.
func (s *Service) OnInit(event *common.ApplicationEvent) {
	err := yaml.Unmarshal(event.Data, s)
	if err == nil {
		s.OnFinish()
		// устанавливаем уровень логирования
		if existsLevel, ok := logLevelByName[s.LevelName]; ok {
			level = existsLevel
		}

		messages = make(chan *Message)
		// заново инициализируем вывод для логов
		writers.init()
		writers.run()
	} else {
		FailExitWithErr(err)
	}
}

// OnRun ничего не делает, писатели логов уже пишут. Заглушка.
func (s *Service) OnRun() {}

// Events не участвует в отправке писем. Заглушка.
func (s *Service) Events() chan *common.SendEvent {
	return nil
}

// OnFinish закрывает канал логирования.
func (s *Service) OnFinish() {
	close(messages)
}
