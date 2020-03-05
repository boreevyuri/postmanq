package connector

import (
	"net"

	"github.com/boreevyuri/postmanq/common"
)

// MailServerStatus статус почтового сервис
type MailServerStatus int

const (
	// LookupMailServerStatus по сервису ведется поиск информации
	LookupMailServerStatus MailServerStatus = iota

	// SuccessMailServerStatus по сервису успешно собрана информация
	SuccessMailServerStatus

	// ErrorMailServerStatus по сервису не удалось собрать информацию
	ErrorMailServerStatus
)

// MailServer почтовый сервис
type MailServer struct {
	// серверы почтового сервиса
	mxServers []*MxServer

	// номер потока, собирающего информацию о почтовом сервисе
	connectorID int

	// статус, говорящий о том, собранали ли информация о почтовом сервисе
	status MailServerStatus
}

// MxServer почтовый сервер
type MxServer struct {
	// доменное имя почтового сервера
	hostname string

	// ip сервера
	ips []net.IP

	// клиенты сервера
	clients []*common.SMTPClient

	// А запись сервера
	realServerName string

	// использоватение TLS
	useTLS bool

	// очередь клиентов
	queues map[string]*common.LimitedQueue
}

// создает новый почтовый сервер
func newMxServer(hostname string) *MxServer {
	queues := make(map[string]*common.LimitedQueue)
	for _, address := range service.Addresses {
		queues[address] = common.NewLimitQueue()
	}

	return &MxServer{
		hostname: hostname,
		ips:      make([]net.IP, 0),
		useTLS:   true,
		queues:   queues,
	}
}

// запрещает использовать TLS соединения
func (m *MxServer) dontUseTLS() {
	m.useTLS = false
}
