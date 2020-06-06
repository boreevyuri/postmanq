package application

import (
	"runtime"

	"github.com/boreevyuri/postmanq/common"
	"github.com/boreevyuri/postmanq/connector"
	"github.com/boreevyuri/postmanq/consumer"
	"github.com/boreevyuri/postmanq/guardian"
	"github.com/boreevyuri/postmanq/limiter"
	"github.com/boreevyuri/postmanq/logger"
	"github.com/boreevyuri/postmanq/mailer"
	"gopkg.in/yaml.v2"
)

// Post приложение, рассылающее письма.
type Post struct {
	Abstract

	// количество отправителей
	Workers int `yaml:"workers"`
}

// NewPost создает новое приложение.
func NewPost() common.Application {
	return new(Post)
}

// Run запускает приложение.
func (p *Post) Run() {
	common.App = p
	//Инициализация сервисов для итератора?
	common.Services = []interface{}{
		guardian.Inst(),
		limiter.Inst(),
		connector.Inst(),
		mailer.Inst(),
	}
	p.services = []interface{}{
		logger.Inst(),
		consumer.Inst(),
		guardian.Inst(),
		limiter.Inst(),
		connector.Inst(),
		mailer.Inst(),
	}
	p.run(p, common.NewApplicationEvent(common.InitApplicationEventKind))
}

// Init инициализирует приложение.
func (p *Post) Init(event *common.ApplicationEvent) {
	// получаем настройки
	err := yaml.Unmarshal(event.Data, p)
	if err == nil {
		p.CommonTimeout.Init()
		common.DefaultWorkersCount = p.Workers

		runtime.GOMAXPROCS(runtime.NumCPU())
		logger.Debug("app workers count %d", p.Workers)
	} else {
		logger.FailExitWithErr(err)
	}
}

// FireRun запускает сервисы приложения.
func (p *Post) FireRun(event *common.ApplicationEvent, abstractService interface{}) {
	service := abstractService.(common.SendingService)
	go service.OnRun()
}

// FireFinish останавливает сервисы приложения.
func (p *Post) FireFinish(event *common.ApplicationEvent, abstractService interface{}) {
	service := abstractService.(common.SendingService)
	go service.OnFinish()
}
