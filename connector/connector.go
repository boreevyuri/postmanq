package connector

import (
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/boreevyuri/postmanq/common"
	"github.com/boreevyuri/postmanq/logger"
)

var (
	connectorEvents = make(chan *ConnectionEvent)
)

// Connector устанавливает соединение к почтовому сервису
type Connector struct {
	// Идентификатор для логов
	id int
}

// создает и запускает новый соединитель
func newConnector(id int) {
	connector := &Connector{id}
	connector.run()
}

// запускает прослушивание событий создания соединений
func (c *Connector) run() {
	for event := range connectorEvents {
		c.connect(event)
	}
}

// устанавливает соединение к почтовому сервису
func (c *Connector) connect(event *ConnectionEvent) {
	logger.Debug("connector#%d-%d try find connection", c.id, event.Message.ID)
	goto receiveConnect

receiveConnect:
	event.TryCount++
	var targetClient *common.SMTPClient

	// смотрим все mx сервера почтового сервиса
	for _, mxServer := range event.server.mxServers {
		logger.Debug("connector#%d-%d try to receive connection for %s", c.id, event.Message.ID, mxServer.hostname)

		// пробуем получить клиента
		event.Queue, _ = mxServer.queues[event.address]
		client := event.Queue.Pop()
		if client != nil {
			targetClient = client.(*common.SMTPClient)
			logger.Debug("connector#%d-%d found free smtp client#%d", c.id, event.Message.ID, targetClient.ID)
		}

		// создаем новое соединение к почтовому сервису
		// если не удалось найти клиента
		// или клиент разорвал соединение
		if (targetClient == nil && !event.Queue.HasLimit()) ||
			(targetClient != nil && targetClient.Status == common.DisconnectedSMTPClientStatus) {
			logger.Debug("connector#%d-%d can't find free smtp client for %s", c.id, event.Message.ID, mxServer.hostname)
			c.createSMTPClient(mxServer, event, &targetClient)
		}

		if targetClient != nil {
			break
		}
	}

	// если клиент не создан, значит мы создали максимум соединений к почтовому сервису
	if targetClient == nil {
		// приостановим работу горутины
		goto waitConnect
	} else {
		targetClient.Wakeup()
		event.Client = targetClient
		// передаем событие отправителю
		event.Iterator.Next().(common.SendingService).Events() <- event.SendEvent
	}
	return

waitConnect:
	if event.TryCount >= common.MaxTryConnectionCount {
		common.ReturnMail(
			event.SendEvent,
			// errors.New(fmt.Sprintf("connector#%d can't connect to %s", c.id, event.Message.HostnameTo)),
			fmt.Errorf("connector#%d can't connect to %s", c.id, event.Message.HostnameTo),
		)
	} else {
		logger.Debug("connector#%d-%d can't find free connections, wait...", c.id, event.Message.ID)
		time.Sleep(common.App.Timeout().Sleep)
		goto receiveConnect
	}
	return
}

// создает соединение к почтовому сервису
func (c *Connector) createSMTPClient(mxServer *MxServer, event *ConnectionEvent, ptrSMTPClient **common.SMTPClient) {
	// устанавливаем ip, с которого будем отсылать письмо
	tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(event.address, "0"))
	if err == nil {
		logger.Debug("connector#%d-%d resolve tcp address %s", c.id, event.Message.ID, tcpAddr.String())
		dialer := &net.Dialer{
			Timeout:   common.App.Timeout().Connection,
			LocalAddr: tcpAddr,
		}
		hostname := net.JoinHostPort(mxServer.hostname, "25")
		// создаем соединение к почтовому сервису
		connection, err := dialer.Dial("tcp", hostname)
		if err == nil {
			logger.Debug("connector#%d-%d connect to %s", c.id, event.Message.ID, hostname)
			connection.SetDeadline(time.Now().Add(common.App.Timeout().Hello))
			client, err := smtp.NewClient(connection, mxServer.hostname)
			if err == nil {
				logger.Debug("connector#%d-%d create client to %s", c.id, event.Message.ID, mxServer.hostname)
				err = client.Hello(service.Domain)
				if err == nil {
					logger.Debug("connector#%d-%d send command HELLO: %s", c.id, event.Message.ID, service.Domain)
					// проверяем доступно ли TLS
					if mxServer.useTLS {
						mxServer.useTLS, _ = client.Extension("STARTTLS")
					}
					logger.Debug("connector#%d-%d use TLS %v", c.id, event.Message.ID, mxServer.useTLS)
					// создаем TLS или обычное соединение
					if mxServer.useTLS {
						c.initTLSSMTPClient(mxServer, event, ptrSMTPClient, connection, client)
					} else {
						c.initSMTPClient(mxServer, event, ptrSMTPClient, connection, client)
					}
				} else {
					client.Quit()
					logger.Debug("connector#%d-%d can't create client to %s, err - %v", c.id, event.Message.ID, mxServer.hostname, err)
				}
			} else {
				// если не удалось создать клиента,
				// возможно, на почтовом сервисе стоит ограничение на количество активных клиентов
				// ставим лимит очереди, чтобы не пытаться открывать новые соединения и не создавать новые клиенты
				event.Queue.HasLimitOn()
				connection.Close()
				logger.Warn("connector#%d-%d can't create client to %s, err - %v", c.id, event.Message.ID, mxServer.hostname, err)
			}
		} else {
			// если не удалось установить соединение,
			// возможно, на почтовом сервисе стоит ограничение на количество соединений
			// ставим лимит очереди, чтобы не пытаться открывать новые соединения
			event.Queue.HasLimitOn()
			logger.Warn("connector#%d-%d can't dial to %s, err - %v", c.id, event.Message.ID, hostname, err)
		}
	} else {
		logger.Warn("connector#%d-%d can't resolve tcp address %s, err - %v", c.id, event.Message.ID, tcpAddr.String(), err)
	}
}

// открывает защищенное соединение
func (c *Connector) initTLSSMTPClient(mxServer *MxServer, event *ConnectionEvent, ptrSMTPClient **common.SMTPClient, connection net.Conn, client *smtp.Client) {
	// если есть какие данные о сертификате и к серверу можно создать TLS соединение
	if mxServer.useTLS {
		// открываем TLS соединение
		err := client.StartTLS(service.getConf(mxServer.realServerName))
		// если все нормально, создаем клиента
		if err == nil {
			c.initSMTPClient(mxServer, event, ptrSMTPClient, connection, client)
		} else {
			// если не удалось создать TLS соединение
			// говорим, что не надо больше создавать TLS соединение
			mxServer.dontUseTLS()
			// разрываем созданое соединение
			// это необходимо, т.к. не все почтовые сервисы позволяют продолжить отправку письма
			// после неудачной попытке создать TLS соединение
			client.Quit()
			// создаем обычное соединие
			c.createSMTPClient(mxServer, event, ptrSMTPClient)

			logger.Warn("connector#%d-%d can't start tls err - %v", c.id, event.Message.ID, err)
		}
	} else {
		c.initSMTPClient(mxServer, event, ptrSMTPClient, connection, client)
	}
}

// создает или инициализирует клиента
func (c *Connector) initSMTPClient(mxServer *MxServer, event *ConnectionEvent, ptrSMTPClient **common.SMTPClient, connection net.Conn, client *smtp.Client) {
	isNil := *ptrSMTPClient == nil
	if isNil {
		var count int
		for _, queue := range mxServer.queues {
			count += queue.MaxLen()
		}
		*ptrSMTPClient = &common.SMTPClient{
			ID: count + 1,
		}
		// увеличиваем максимальную длину очереди
		event.Queue.AddMaxLen()
	}
	smtpClient := *ptrSMTPClient
	smtpClient.Conn = connection
	smtpClient.Worker = client
	smtpClient.ModifyDate = time.Now()
	if isNil {
		logger.Debug("connector#%d-%d create smtp client#%d for %s", c.id, event.Message.ID, smtpClient.ID, mxServer.hostname)
	} else {
		logger.Debug("connector#%d-%d reopen smtp client#%d for %s", c.id, event.Message.ID, smtpClient.ID, mxServer.hostname)
	}
}
