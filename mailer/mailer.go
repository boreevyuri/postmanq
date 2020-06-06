package mailer

import (
	"fmt"

	"github.com/boreevyuri/dkim"
	"github.com/boreevyuri/postmanq/common"
	"github.com/boreevyuri/postmanq/logger"
)

// Mailer отправитель письма.
type Mailer struct {
	// идентификатор для логов
	id int
}

// создает нового отправителя.
func newMailer(id int) {
	mailer := &Mailer{id}
	mailer.run()
}

// запускает отправителя.
func (m *Mailer) run() {
	for event := range events {
		m.sendMail(event)
	}
}

// sendMail вписывает dkim и отправляет письмо в созданное ранее соединение.
func (m *Mailer) sendMail(event *common.SendEvent) {
	message := event.Message
	if common.EmailRegexp.MatchString(message.Envelope) && common.EmailRegexp.MatchString(message.Recipient) {
		m.signWithDKIM(message)
		//if err := m.newSend(event); err != nil {
		//	event.Client.Wait()
		//	event.Queue.Push(event.Client)
		//	common.ReturnMail(event, err)
		//}
		m.send(event)
	} else {
		var invalidAddressError = "511 service#%d can't send mail#%d, envelope or recipient is invalid"
		common.ReturnMail(event, fmt.Errorf(invalidAddressError, m.id, message.ID))
	}
}

// signWithDKIM создает dkim-подпись и добавляет заголовок Dkim в письмо.
func (m *Mailer) signWithDKIM(message *common.MailMessage) {
	conf, err := dkim.NewConf(message.HostnameFrom, service.DkimSelector)
	if err != nil {
		logger.Warn("mailer#%d-%d can't create dkim config, error - %v", m.id, message.ID, err)
	} else {
		conf[dkim.AUIDKey] = message.Envelope
		conf[dkim.CanonicalizationKey] = "relaxed/relaxed"
		signer := dkim.NewByKey(conf, service.privateKey)
		//if err == nil {
		signed, err := signer.Sign([]byte(message.Body))
		if err == nil {
			message.Body = string(signed)
			logger.Debug("mailer#%d-%d success sign mail", m.id, message.ID)
		} else {
			logger.Warn("mailer#%d-%d can't sign mail, error - %v", m.id, message.ID, err)
		}
		//} else {
		//	logger.Warn("mailer#%d-%d can't create dkim signer, error - %v", m.id, message.ID, err)
		//}
	}
}

func (m *Mailer) newSend(event *common.SendEvent) error {
	message := event.Message
	worker := event.Client.Worker
	mailerID := "mailer#" + string(m.id) + "-" + string(message.ID)

	logger.Info(mailerID + " begin sending mail")
	logger.Debug(mailerID+" receive SMTP client#%d", event.Client.ID)

	event.Client.SetTimeout(common.App.Timeout().Mail)

	if err := worker.Mail(message.Envelope); err != nil {
		logger.Info(mailerID+" error after MAIL FROM: %+v", err)
		return err
	}

	event.Client.SetTimeout(common.App.Timeout().Rcpt)

	if err := worker.Rcpt(message.Recipient); err != nil {
		logger.Info(mailerID+" error after RCPT TO: %+v", err)
		return err
	}

	event.Client.SetTimeout(common.App.Timeout().Data)

	wc, err := worker.Data()
	if err != nil {
		logger.Info(mailerID+" error after DATA: %+v", err)
		return err
	}

	if _, err = fmt.Fprint(wc, message.Body); err != nil {
		logger.Info(mailerID+" error during body send: %+v", err)
		return err
	}

	if err = wc.Close(); err != nil {
		logger.Info(mailerID+" error after sent data: %+v", err)
		return err
	}

	logger.Info(mailerID+" success sent mail for %s", message.Recipient)
	event.Result <- common.SuccessSendEventResult

	if err = worker.Reset(); err != nil {
		logger.Info(mailerID+" error after RSET. Error:%+v", err)
		return err
	}

	//Отпускаем SMTPClient в очередь
	event.Client.Wait()
	event.Queue.Push(event.Client)

	return nil
}

// send отправляет письмо.
func (m *Mailer) send(event *common.SendEvent) {
	message := event.Message
	worker := event.Client.Worker

	logger.Info("mailer#%d-%d begin sending mail", m.id, message.ID)
	logger.Debug("mailer#%d-%d receive smtp client#%d", m.id, message.ID, event.Client.ID)

	success := false

	var sendErr error = nil

	event.Client.SetTimeout(common.App.Timeout().Mail)

	err := worker.Mail(message.Envelope)
	if err == nil {
		logger.Debug("mailer#%d-%d sent command MAIL FROM: %s", m.id, message.ID, message.Envelope)

		event.Client.SetTimeout(common.App.Timeout().Rcpt)

		err = worker.Rcpt(message.Recipient)
		if err == nil {
			logger.Debug("mailer#%d-%d sent command RCPT TO: %s", m.id, message.ID, message.Recipient)

			event.Client.SetTimeout(common.App.Timeout().Data)

			wc, err := worker.Data()
			if err == nil {
				logger.Debug("mailer#%d-%d sent command DATA", m.id, message.ID)

				_, err = fmt.Fprint(wc, message.Body)
				if err == nil {
					err = wc.Close()
					if err == nil {
						// logger.Debug("%s", message.Body)
						logger.Debug("mailer#%d-%d body sent successful. Sent command . ", m.id, message.ID)

						// Успешная отправка. Не закрываем соединение, но отсылаем RSET.
						err = worker.Reset()
						if err == nil {
							logger.Debug("mailer#%d-%d sent command RSET", m.id, message.ID)
							logger.Info("mailer#%d-%d success send mail for %s", m.id, message.ID, message.Recipient)

							success = true
						} else {
							logger.Info("mailer#%d-%d error after RSET. Error: %+v", m.id, message.ID, err)
							sendErr = err
						}
					} else {
						logger.Info("mailer#%d-%d error after sent body. Error: %+v", m.id, message.ID, err)
						sendErr = err
					}
				} else {
					logger.Info("mailer#%d-%d error during body send. Error: %+v", m.id, message.ID, err)
					sendErr = err
				}
			} else {
				logger.Info("mailer#%d-%d error after DATA. Error: %+v", m.id, message.ID, err)
				sendErr = err
			}
		} else {
			logger.Info("mailer#%d-%d error after RCPT TO. Error: %+v", m.id, message.ID, err)
			sendErr = err
		}
	} else {
		logger.Info("mailer#%d-%d error after MAIL FROM. Error: %+v", m.id, message.ID, err)
		sendErr = err
	}

	event.Client.Wait()
	event.Queue.Push(event.Client)

	if success {
		// отпускаем поток получателя сообщений из очереди
		event.Result <- common.SuccessSendEventResult
	} else {
		common.ReturnMail(event, sendErr)
	}
}
