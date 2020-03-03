package guardian

import (
	"github.com/boreevyuri/postmanq/common"
	"github.com/boreevyuri/postmanq/logger"
	yaml "gopkg.in/yaml.v2"
)

var (
	// сервис блокирующий отправку писем
	service *Service

	// канал для приема событий отправки писем
	events = make(chan *common.SendEvent)
)

// Service сервис, блокирующий отправку писем
type Service struct {
	// хосты, на которые блокируется отправка писем
	Hostnames []string `yaml:"exclude"`

	// длина массива хостов, необходима для поиска
	hostnameLen int

	// количество горутин блокирующий отправку писем к почтовым сервисам
	GuardiansCount int `yaml:"workers"`
}

// Inst создает новый сервис блокировок
func Inst() common.SendingService {
	if service == nil {
		service = new(Service)
	}
	return service
}

// OnInit инициализирует сервис блокировок
func (s *Service) OnInit(event *common.ApplicationEvent) {
	logger.Debug("init guardians...")
	err := yaml.Unmarshal(event.Data, s)
	if err == nil {
		s.hostnameLen = len(s.Hostnames)
		if s.GuardiansCount == 0 {
			s.GuardiansCount = common.DefaultWorkersCount
		}
	} else {
		logger.FailExitWithErr(err)
	}
}

// OnRun запускает горутины
func (s *Service) OnRun() {
	for i := 0; i < s.GuardiansCount; i++ {
		go newGuardian(i + 1)
	}
}

// Events канал для приема событий отправки писем
func (s *Service) Events() chan *common.SendEvent {
	return events
}

// OnFinish завершает работу сервиса соединений
func (s *Service) OnFinish() {
	close(events)
}
