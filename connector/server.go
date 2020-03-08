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

// MxServer найденный seeker-ом почтовый сервер получателя
type MxServer struct {
	// доменное имя почтового сервера получателя
	hostname string

	// IP-адреса сервера получателя
	ips []net.IP

	// клиенты сервера
	clients []*common.SMTPClient

	// PTR запись сервера получателя
	realServerName string

	// использует ли сервер получателя TLS
	useTLS bool

	// очередь клиентов к серверу получателя
	queues map[string]*common.LimitedQueue
}

// newMxServer добавляет в карту новый почтовый сервер получателя
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
