package common

import (
	"net"
	"net/smtp"
	"time"
)

// SMTPClientStatus статус клиента почтового сервера
type SMTPClientStatus int

const (
	// WorkingSMTPClientStatus отсылает письмо
	WorkingSMTPClientStatus SMTPClientStatus = iota

	// WaitingSMTPClientStatus ожидает письма
	WaitingSMTPClientStatus

	// DisconnectedSMTPClientStatus отсоединен
	DisconnectedSMTPClientStatus
)

// SMTPClient клиент почтового сервера
type SMTPClient struct {
	// идентификатор клиента для удобства в логах
	ID int

	// соединение к почтовому серверу
	Conn net.Conn

	// реальный smtp клиент
	Worker *smtp.Client

	// дата создания или изменения статуса клиента
	ModifyDate time.Time

	// статус SMTPClient
	Status SMTPClientStatus

	// таймер, по истечении которого, соединение к почтовому сервису будет разорвано
	timer *time.Timer
}

// SetTimeout устанавливайт таймаут на чтение и запись соединения
func (s *SMTPClient) SetTimeout(timeout time.Duration) {
	s.Conn.SetDeadline(time.Now().Add(timeout))
}

// Close принудительно закрывает соединение
// mail.ru обрывает соединение со своей стороны, получаем broken pipe
func (s *SMTPClient) Close() {
	s.Status = DisconnectedSMTPClientStatus
	s.Worker.Close()
	s.timer = nil
}

// Wait переводит клиента в ожидание
// после окончания ожидания соединение разрывается, а статус меняется на отсоединенный
func (s *SMTPClient) Wait() {
	s.Status = WaitingSMTPClientStatus
	s.timer = time.AfterFunc(App.Timeout().Waiting, func() {
		s.Status = DisconnectedSMTPClientStatus
		s.Worker.Close()
		s.timer = nil
	})
}

// Wakeup переводит клиента в рабочее состояние
// если клиент был в ожидании, ожидание прерывается
func (s *SMTPClient) Wakeup() {
	s.Status = WorkingSMTPClientStatus
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}
