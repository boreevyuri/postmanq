package postmanq

import (
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"time"
)

// уровень логирования
type LogLevel int

// уровни логирования
const (
	LOG_LEVEL_DEBUG LogLevel = iota
	LOG_LEVEL_INFO
	LOG_LEVEL_WARNING
	LOG_LEVEL_ERROR
)

// название уровней логирования
const (
	LOG_LEVEL_DEBUG_NAME   = "debug"
	LOG_LEVEL_INFO_NAME    = "info"
	LOG_LEVEL_WARNING_NAME = "warning"
	LOG_LEVEL_ERROR_NAME   = "error"
)

type LoggerStatus int

const (
	LOGGER_STATUS_ACTIVE LoggerStatus = iota
	LOGGER_STATUS_WAIT
)

var (
	filenameRegex = regexp.MustCompile(`[^\\/]+\.[^\\/]+`)
	// названия уровней логирования, используется непосредственно в момент создания записи в лог
	logLevelById = map[LogLevel]string{
		LOG_LEVEL_DEBUG:   LOG_LEVEL_DEBUG_NAME,
		LOG_LEVEL_INFO:    LOG_LEVEL_INFO_NAME,
		LOG_LEVEL_WARNING: LOG_LEVEL_WARNING_NAME,
		LOG_LEVEL_ERROR:   LOG_LEVEL_ERROR_NAME,
	}
	// уровни логирования по названию, используется для удобной инициализации сервиса логирования
	logLevelByName = map[string]LogLevel{
		LOG_LEVEL_DEBUG_NAME:   LOG_LEVEL_DEBUG,
		LOG_LEVEL_INFO_NAME:    LOG_LEVEL_INFO,
		LOG_LEVEL_WARNING_NAME: LOG_LEVEL_WARNING,
		LOG_LEVEL_ERROR_NAME:   LOG_LEVEL_ERROR,
	}
	logger *Logger
)

// запись логирования
type LogMessage struct {
	Message string        // сообщение для лога, может содержать параметры
	Level   LogLevel      // уровень логирования записи, необходим для отсечения лишних записей
	Args    []interface{} // аргументы для параметров сообщения
}

// созадние новой записи логирования
func NewLogMessage(level LogLevel, message string, args ...interface{}) *LogMessage {
	logMessage := new(LogMessage)
	logMessage.Level = level
	logMessage.Message = message
	logMessage.Args = args
	return logMessage
}

type LogWriter interface {
	WriteString(string)
}

type StdoutWriter struct{}

func (this *StdoutWriter) WriteString(str string) {
	os.Stdout.WriteString(str)
}

type FileWriter struct {
	filename string
}

func (this *FileWriter) WriteString(str string) {
	f, err := os.OpenFile(this.filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err == nil {
		_, err = f.WriteString(str)
		f.Close()
	}
}

// сервис логирования
type Logger struct {
	LogLevelName string           `yaml:"logLevel"`  // название уровня логирования, устанавливается в конфиге
	Output       string           `yaml:"logOutput"` // название вывода логов
	level        LogLevel         // уровень логов, ниже этого уровня логи писаться не будут
	writer       LogWriter        // куда пишем логи stdout или файл
	messages     chan *LogMessage // канал логирования
}

// создает новый сервис логирования
func LoggerOnce() *Logger {
	if logger == nil {
		logger = new(Logger)
		// инициализируем сервис с настройками по умолчанию
		logger.messages = make(chan *LogMessage)
		logger.level = LOG_LEVEL_WARNING
		// запускаем запись логов в отдельном потоке
		for i := 0; i < defaultWorkersCount; i++ {
			go logger.write(i)
		}
		logger.initWriter()
	}
	return logger
}

// инициализирует сервис логирования
func (this *Logger) OnInit(event *ApplicationEvent) {
	err := yaml.Unmarshal(event.Data, this)
	if err == nil {
		// устанавливаем уровень логирования
		if level, ok := logLevelByName[this.LogLevelName]; ok {
			this.level = level
		}
		// заново инициализируем вывод для логов
		this.initWriter()
	} else {
		FailExitWithErr(err)
	}
}

func (this *Logger) OnRun() {}

// закрывает канал логирования
func (this *Logger) OnFinish() {
	close(this.messages)
}

// пишет логи в вывод в отдельном потоке
func (this *Logger) write(id int) {
	for message := range this.messages {
		this.writeMessage(id, message)
	}
}

func (this *Logger) writeMessage(id int, message *LogMessage) {
	if this.writer != nil {
		this.writer.WriteString(
			fmt.Sprintf(
				"PostmanQ | %v | %s: %s\n",
				time.Now().Format("2006-01-02 15:04:05"),
				logLevelById[message.Level],
				fmt.Sprintf(message.Message, message.Args...),
			),
		)
	}
}

// инициализирует вывод логирования
func (this *Logger) initWriter() {
	this.writer = nil
	if filenameRegex.MatchString(this.Output) { // проверяем получили ли из настроек имя файла
		// получаем директорию, в которой лежит файл
		dir := filepath.Dir(this.Output)
		// смотрим, что она реально существует
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			FailExit("directory %s is not exists", dir)
		} else {
			this.writer = &FileWriter{this.Output}
		}
	} else if len(this.Output) == 0 || this.Output == "stdout" {
		this.writer = new(StdoutWriter)
	}
}

// посылает сервису логирования запись для логирования произвольного уровня
func log(message string, level LogLevel, args ...interface{}) {
	defer func() { recover() }()
	// если уровень записи не ниже уровня сервиса логирования
	// запись посылается сервису
	if logger.level <= level {
		// если уровень выше "info", значит пишется ошибка
		// добавляем к сообщению стек, чтобы посмотреть в чем дело
		if level > LOG_LEVEL_INFO && logger.level == LOG_LEVEL_DEBUG {
			message = fmt.Sprint(message, "\n", string(debug.Stack()))
		}
		logger.messages <- NewLogMessage(level, message, args...)
	}
}

// пишет ошибку в лог
func Err(message string, args ...interface{}) {
	log(message, LOG_LEVEL_ERROR, args...)
}

// пишет произвольную ошибку в лог и завершает программу
func FailExit(message string, args ...interface{}) {
	Err(message, args...)
	app.Events() <- NewApplicationEvent(FinishApplicationEventKind)
}

// пишет системную ошибку в лог и завершает программу
func FailExitWithErr(err error) {
	FailExit("%v", err)
}

// пишет произвольное предупреждение
func Warn(message string, args ...interface{}) {
	log(message, LOG_LEVEL_WARNING, args...)
}

// пишет системное предупреждение
func WarnWithErr(err error) {
	Warn("%v", err)
}

// пишет информационное сообщение
func Info(message string, args ...interface{}) {
	log(message, LOG_LEVEL_INFO, args...)
}

// пишет сообщение для отладки
func Debug(message string, args ...interface{}) {
	log(message, LOG_LEVEL_DEBUG, args...)
}
