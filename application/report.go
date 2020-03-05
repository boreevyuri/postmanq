package application

import (
	"github.com/boreevyuri/postmanq/analyser"
	"github.com/boreevyuri/postmanq/common"
	"github.com/boreevyuri/postmanq/consumer"
)

// Report приложение, анализирующее неотправленные сообщения
type Report struct {
	Abstract
}

// NewReport создает новое приложение
func NewReport() common.Application {
	return new(Report)
}

// Run запускает приложение
func (r *Report) Run() {
	common.App = r
	common.Services = []interface{}{
		analyser.Inst(),
	}
	r.services = []interface{}{
		consumer.Inst(),
		analyser.Inst(),
	}
	r.run(r, common.NewApplicationEvent(common.InitApplicationEventKind))
}

// FireRun запускает сервисы приложения
func (r *Report) FireRun(event *common.ApplicationEvent, abstractService interface{}) {
	service := abstractService.(common.ReportService)
	go service.OnShowReport()
}
