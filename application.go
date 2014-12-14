package postmanq

import (
	"runtime"
	"io/ioutil"
	"time"
	"crypto/x509"
)

const (
	EXAMPLE_CONFIG_YAML = "/path/to/config/file.yaml"
)

var (
	app *Application
)

// Событие инициализации. Вызывается после регистрации сервисов и содержит данные для настройки сервисов.
type InitEvent struct {
	Data         []byte // данные с настройками каждого сервиса
	MailersCount int    // количество потоков, отправляющих письма
}

// Программа отправки почты получилась довольно сложной, т.к. она выполняет обработку и отправку писем,
// работает с диском и с сетью, ведет логирование и проверяет ограничения перед отправкой.
// Из - за такого насыщенного функционала, было принято решение разбить программу на логические части - сервисы.
// Сервис - это модуль программы, отвечающий за выполнение одной конкретной задачи, например логирование.
// Сервис может сам выполнять эту задачу, либо управлять выполнением задачи.
// Сервисы взаимодействуют через события.
type Service interface {
	OnRegister()
	OnInit(*InitEvent)
	OnRun()
	OnFinish()
}

// Сервис, участвующий в отправке письма.
type SendService interface {
	OnSend(*SendEvent)
}

// Событие отправки письма
type SendEvent struct {
	Client           *SmtpClient    // объект, содержащий подключение и клиент для отправки писем
	CertPool         *x509.CertPool // пул сертификатов
	CertBytes        []byte         // для каждого почтового сервиса необходим подписывать сертификат, поэтому в событии храним сырые данные сертификата
	CertBytesLen     int            // длина сертификата, по если длина больше 0, тогда пытаемся отправлять письма через TLS
	Message          *MailMessage   // само письмо, полученное из очереди
	DefaultPrevented bool           // флаг, сигнализирующий обрабатывать ли событие
	CreateDate       time.Time      // дата создания необходима при получении подключения к почтовому сервису
}

// тип гловального события приложения
type ApplicationEventKind int

const (
	APPLICATION_EVENT_KIND_REGISTER ApplicationEventKind = iota // событие регистрации сервисов
	APPLICATION_EVENT_KIND_INIT                                 // событие инициализации сервисов
	APPLICATION_EVENT_KIND_RUN                                  // событие запуска сервисов
	APPLICATION_EVENT_KIND_FINISH                               // событие завершения сервисов
)

// событие приложения
type ApplicationEvent struct {
	kind ApplicationEventKind
}

// создает событие с указанным типом
func NewApplicationEvent(kind ApplicationEventKind) *ApplicationEvent {
	return &ApplicationEvent{kind: kind}
}

// приложение
type Application struct {
	ConfigFilename string                          // путь до конфигурационного файла
	services       []Service                       // сервисы приложения
	events         chan *ApplicationEvent          // канал событий приложения
	done           chan bool                       // флаг, сигнализирующий окончание работы приложения
	handlers       map[ApplicationEventKind]func() // обработчики событий приложения
}

// создает приложение
func NewApplication() *Application {
	if app == nil {
		app = new(Application)
		// инициализируем сервисы приложения
		app.services = []Service{
			NewLogger(),
			NewLimiter(),
			NewConnector(),
			NewMailer(),
			NewConsumer(),
		}
		// создаем каналы для событий
		app.events = make(chan *ApplicationEvent, 4)
		// и ожидания окончания программы
		app.done = make(chan bool)
		// устанавливаем обработчики для событий
		app.handlers = map[ApplicationEventKind]func(){
			APPLICATION_EVENT_KIND_REGISTER: app.registerServices,
			APPLICATION_EVENT_KIND_INIT    : app.initServices,
			APPLICATION_EVENT_KIND_RUN     : app.runServices,
			APPLICATION_EVENT_KIND_FINISH  : app.finishServices,
		}
	}
	return app
}

// регистрирует сервисы приложения
func (this *Application) registerServices() {
	for _, service := range this.services {
		service.OnRegister()
	}
	// отсылаем событие инициализации
	this.events <- NewApplicationEvent(APPLICATION_EVENT_KIND_INIT)
}

// инициализирует сервисы приложения
func (this *Application) initServices() {
	// проверяем, установлен ли конфигурационный файл
	if len(this.ConfigFilename) > 0 && this.ConfigFilename != EXAMPLE_CONFIG_YAML {
		// пытаемся прочитать конфигурационный файл
		bytes, err := ioutil.ReadFile(this.ConfigFilename)
		if err == nil {
			// создаем событие инициализации
			event := new(InitEvent)
			event.Data = bytes
			// и оповещаем сервисы
			for _, service := range this.services {
				service.OnInit(event)
			}
			this.events <- NewApplicationEvent(APPLICATION_EVENT_KIND_RUN)
		} else {
			FailExitWithErr(err)
		}
	} else {
		FailExit("configuration file not found")
	}
}

// запускает приложения
func (this *Application) runServices() {
	for _, service := range this.services {
		go service.OnRun()
	}
}

// останавливает приложения
func (this *Application) finishServices() {
	for _, service := range this.services {
		service.OnFinish()
	}
	this.done <- true
}

// запускает и контролирует работу всего приложения
func (this *Application) Run() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	go func() {
		for {
			select {
			case event := <- this.events:
				if handler, ok := this.handlers[event.kind]; ok {
					handler()
				}
			}
		}
		close(this.events)
	}()
	this.events <- NewApplicationEvent(APPLICATION_EVENT_KIND_REGISTER)
	<- this.done
}
