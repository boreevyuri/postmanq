package application

import (
	"io/ioutil"
	"time"

	"github.com/boreevyuri/postmanq/common"
	"github.com/boreevyuri/postmanq/logger"
)

// Abstract базовое приложение
type Abstract struct {
	// путь до конфигурационного файла
	configFilename string

	// сервисы приложения, отправляющие письма
	services []interface{}

	// канал событий приложения
	events chan *common.ApplicationEvent

	// флаг, сигнализирующий окончание работы приложения
	done chan bool

	CommonTimeout common.Timeout `yaml:"timeouts"`
}

// IsValidConfigFilename проверяет валидность пути к файлу с настройками
func (a *Abstract) IsValidConfigFilename(filename string) bool {
	return len(filename) > 0 && filename != common.ExampleConfigYaml
}

// запускает основной цикл приложения
func (a *Abstract) run(app common.Application, event *common.ApplicationEvent) {
	app.SetDone(make(chan bool))
	// создаем каналы для событий
	app.SetEvents(make(chan *common.ApplicationEvent, 3))
	go func() {
		for {
			select {
			case event := <-app.Events():
				if event.Kind == common.InitApplicationEventKind {
					// пытаемся прочитать конфигурационный файл
					bytes, err := ioutil.ReadFile(a.configFilename)
					if err == nil {
						event.Data = bytes
						app.Init(event)
					} else {
						logger.FailExit("application can't read configuration file, error -  %v", err)
					}
				}

				for _, service := range app.Services() {
					switch event.Kind {
					case common.InitApplicationEventKind:
						app.FireInit(event, service)
					case common.RunApplicationEventKind:
						app.FireRun(event, service)
					case common.FinishApplicationEventKind:
						app.FireFinish(event, service)
					}
				}

				switch event.Kind {
				case common.InitApplicationEventKind:
					event.Kind = common.RunApplicationEventKind
					app.Events() <- event
				case common.FinishApplicationEventKind:
					time.Sleep(30 * time.Second)
					app.Done() <- true
				}
			}
		}
		close(app.Events())
	}()
	app.Events() <- event
	<-app.Done()
}

// SetConfigFilename устанавливает путь к файлу с настройками
func (a *Abstract) SetConfigFilename(configFilename string) {
	a.configFilename = configFilename
}

// SetEvents устанавливает канал событий приложения
func (a *Abstract) SetEvents(events chan *common.ApplicationEvent) {
	a.events = events
}

// Events возвращает канал событий приложения
func (a *Abstract) Events() chan *common.ApplicationEvent {
	return a.events
}

// SetDone устанавливает канал завершения приложения
func (a *Abstract) SetDone(done chan bool) {
	a.done = done
}

// Done возвращает канал завершения приложения
func (a *Abstract) Done() chan bool {
	return a.done
}

// Services возвращает сервисы, используемые приложением
func (a *Abstract) Services() []interface{} {
	return a.services
}

// FireInit инициализирует сервисы
func (a *Abstract) FireInit(event *common.ApplicationEvent, abstractService interface{}) {
	service := abstractService.(common.Service)
	service.OnInit(event)
}

// Init инициализирует приложение
func (a *Abstract) Init(event *common.ApplicationEvent) {}

// Run запускает приложение
func (a *Abstract) Run() {}

// RunWithArgs запускает приложение с аргументами
func (a *Abstract) RunWithArgs(args ...interface{}) {}

// FireRun запускает сервисы приложения
func (a *Abstract) FireRun(event *common.ApplicationEvent, abstractService interface{}) {}

// FireFinish останавливает сервисы приложения
func (a *Abstract) FireFinish(event *common.ApplicationEvent, abstractService interface{}) {}

// Timeout возвращает таймауты приложения
func (a *Abstract) Timeout() common.Timeout {
	return a.CommonTimeout
}
