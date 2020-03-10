package common

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// MaxTryConnectionCount Максимальное количество попыток подключения к почтовику за отправку письма
	MaxTryConnectionCount int = 30
	// MaxSendingCount Максимальное количество попыток отправки письма
	MaxSendingCount int = 96
)

var (
	// EmailRegexp Регулярка для проверки адреса почты, сразу компилируем, чтобы при отправке не терять на этом время
	EmailRegexp = regexp.MustCompile(`^[\w\d\.\_\%\+\-]+@([\w\d\.\-]+\.\w{2,4})$`)
)

// Timeout таймауты приложения
type Timeout struct {
	Sleep      time.Duration `yaml:"sleep"`
	Waiting    time.Duration `yaml:"waiting"`
	Connection time.Duration `yaml:"connection"`
	Hello      time.Duration `yaml:"hello"`
	Mail       time.Duration `yaml:"mail"`
	Rcpt       time.Duration `yaml:"rcpt"`
	Data       time.Duration `yaml:"data"`
}

// Init инициализирует значения таймаутов по умолчанию
func (t *Timeout) Init() {
	if t.Sleep == 0 {
		t.Sleep = time.Second
	}
	if t.Waiting == 0 {
		t.Waiting = 30 * time.Second
	}
	if t.Connection == 0 {
		t.Connection = 5 * time.Minute
	}
	if t.Hello == 0 {
		t.Hello = 5 * time.Minute
	}
	if t.Mail == 0 {
		t.Mail = 5 * time.Minute
	}
	if t.Rcpt == 0 {
		t.Rcpt = 5 * time.Minute
	}
	if t.Data == 0 {
		t.Data = 10 * time.Minute
	}
}

// DelayedBindingType тип отложенной очереди
type DelayedBindingType int

const (
	// UnknownDelayedBinding хз пока что это
	UnknownDelayedBinding DelayedBindingType = iota
	// SecondDelayedBinding биндинг откладывания на секунду
	SecondDelayedBinding
	// ThirtySecondDelayedBinding отложить на 30 сек
	ThirtySecondDelayedBinding
	// MinuteDelayedBinding отложить на минуту
	MinuteDelayedBinding
	// FiveMinutesDelayedBinding отложить на 5 минут
	FiveMinutesDelayedBinding
	// TenMinutesDelayedBinding отложить на 10 минут
	TenMinutesDelayedBinding
	// TwentyMinutesDelayedBinding отложить на 20 минут
	TwentyMinutesDelayedBinding
	// ThirtyMinutesDelayedBinding отложить на 30 минут
	ThirtyMinutesDelayedBinding
	// FortyMinutesDelayedBinding отложить на 40 минут
	FortyMinutesDelayedBinding
	// FiftyMinutesDelayedBinding отложить на 50 минут
	FiftyMinutesDelayedBinding
	// HourDelayedBinding отложить на час
	HourDelayedBinding
	// SixHoursDelayedBinding отложить на 6 часов
	SixHoursDelayedBinding
	// DayDelayedBinding отложить на 1 день
	DayDelayedBinding
	// NotSendDelayedBinding отложить в неотправленные
	NotSendDelayedBinding
)

// MailError ошибка во время отпрвки письма
type MailError struct {
	// сообщение об ошибке
	Message string `json:"message"`

	// код ошибки
	Code int `json:"code"`
}

// MailMessage письмо
type MailMessage struct {
	// идентификатор для логов
	ID int64 `json:"-"`

	// отправитель из "envelope" из очереди
	Envelope string `json:"envelope"`

	// получатель "recipient" из очереди
	Recipient string `json:"recipient"`

	// тело письма "body" из очереди
	Body string `json:"body"`

	// домен отправителя, удобно сразу получить и использовать далее
	HostnameFrom string `json:"-"`

	// Домен получателя, удобно сразу получить и использовать далее
	HostnameTo string `json:"-"`

	// дата создания, используется в основном сервисом ограничений
	CreatedDate time.Time `json:"-"`

	// тип очереди, в которою письмо уже было отправлено после неудачной отправки, ипользуется для цепочки очередей
	BindingType DelayedBindingType `json:"bindingType"`

	// ошибка отправки "error" из очереди
	Error *MailError `json:"error"`

	// количество попыток отправки "trySendingCount" из очереди
	TrySendingCount int `json:"trySendingCount"`
}

// Init инициализирует письмо
func (m *MailMessage) Init() {
	m.ID = time.Now().UnixNano()
	m.TrySendingCount++
	m.CreatedDate = time.Now()
	if hostname, err := m.getHostnameFromEmail(m.Envelope); err == nil {
		m.HostnameFrom = hostname
	}
	if hostname, err := m.getHostnameFromEmail(m.Recipient); err == nil {
		m.HostnameTo = hostname
	}
}

// получает домен из адреса "user@domain"
func (m *MailMessage) getHostnameFromEmail(email string) (string, error) {
	matches := EmailRegexp.FindAllStringSubmatch(email, -1)
	if len(matches) == 1 && len(matches[0]) == 2 {
		return matches[0][1], nil
	}
	return "", errors.New("invalid email address")
}

// ReturnMail возвращает письмо обратно в очередь после ошибки во время отправки
func ReturnMail(event *SendEvent, err error) {
	// необходимо проверить сообщение на наличие кода ошибки
	// обычно код идет первым
	if err != nil {
		errorMessage := err.Error()
		parts := strings.Split(errorMessage, " ")
		if len(parts) > 0 {
			// пытаемся получить код
			code, e := strconv.Atoi(strings.TrimSpace(parts[0]))
			errors.New(fmt.Sprintf("ReturnMail catch error - %v", code))
			// и создать ошибку
			// письмо с ошибкой вернется в другую очередь, отличную от письмо без ошибки
			if e == nil {
				event.Message.Error = &MailError{errorMessage, code}
			}
		}
	}

	// если в событии уже создан клиент
	if event.Client != nil {
		if event.Client.Worker != nil {
			// сбрасываем цепочку команд к почтовому сервису
			_ = event.Client.Worker.Reset()
		}
	}

	// отпускаем поток получателя сообщений из очереди
	if event.Message.Error == nil {
		event.Result <- DelaySendEventResult
	} else {
		if event.Message.Error.Code >= 400 && event.Message.Error.Code < 500 {
			event.Result <- DelaySendEventResult
		} else {
			event.Result <- ErrorSendEventResult
		}
	}
}
