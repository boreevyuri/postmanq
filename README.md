# PostmanQ

PostmanQ - это высокопроизводительный почтовый сервер(MTA). 
На сервере под управлением Ubuntu 12.04 с 8-ми ядерным процессором и 32ГБ оперативной памяти 
PostmanQ рассылает более 300 писем в секунду.

Для работы PostmanQ потребуется AMQP-сервер, в котором будут храниться письма. 

PostmanQ разбирает одну или несколько очередей одного или нескольких AMQP-серверов с письмами и отправляет письма по SMTP сторонним почтовым сервисам.

## Возможности

1. PostmanQ может работать с несколькими AMQP-серверами и очередями каждого из серверов.
2. PostmanQ умеет работать через TLS соединение.
3. PostmanQ рассылает письма с разных IP.
4. PostmanQ подписывает DKIM для каждого письма.
5. PostmanQ следит за количеством отправленных писем почтовому сервису.
6. PostmanQ попробует отослать письмо попозже, если возникла сетевая ошибка, письмо попало в [серый список](http://ru.wikipedia.org/wiki/%D0%A1%D0%B5%D1%80%D1%8B%D0%B9_%D1%81%D0%BF%D0%B8%D1%81%D0%BE%D0%BA) или количество отправленных писем почтовому сервису уже максимально.
7. PostmanQ положит в отдельную очередь письма, которые не удалось отправить из-за 5ХХ ошибки

## Как это работает?

1. Нам потребуется AMQP-сервер, например [RabbitMQ](https://www.rabbitmq.com), и [go](http://golang.org/) для компиляции PostmanQ.
2. Выполняем предварительную подготовку и установку PostmanQ. Инструкции описаны ниже.
3. Запускаем RabbitMQ и PostmanQ.
4. Создаем в RabbitMQ одну или несколько очередей.
5. Кладем в очередь письмо. Письмо должно быть в формате json и иметь вид
    
        {
            "envelope": "sender@mail.foo",
            "recipient": "recipient@mail.foo",
            "body": "письмо с заголовками и содержимым"
        }
    
6. PostmanQ забирает письмо из очереди.
7. Проверяет ограничение на количество отправленных писем для почтового сервиса.
8. Открывает TLS или обычное соединение.
9. Создает DKIM.
10. Отправляет письмо стороннему почтовому сервису.
11. Если произошла сетевая ошибка, то письмо перекладывается в одну из очередей для повторной отправки.
12. Если произошла 5ХХ ошибка, то письмо перекладывается в очередь с проблемными письмами, повторная отправка не производится.

## Предварительная подготовка

Чтобы наши письма отправлялись безопасно и доходили до адресатов, не попадая в спам, нам необходимо создать сертификат, публичный и закрытый ключ для каждого домена.

Закрытый ключ будет использоваться для подписи DKIM. 

Публичный ключ необходимо указать в DNS записи для того, чтобы сторонние почтовые сервисы могли валидировать DKIM наших писем.

Сертификат будет использоваться для создания TLS соединений к удаленным почтовым сервисами.

    cd /some/path
    # создаем корневой ключ
    openssl genrsa -out rootCA.key 2048 
    # создаем корневой сертификат на 10000 дней
    openssl req -x509 -new -key rootCA.key -days 10000 -out rootCA.crt
    # создаем приватный ключ
    openssl genrsa -out private.key 2048
    # создаем запрос на сертификат
    openssl req -new -key private.key -out request.csr
    # подписываем сертификат
    openssl x509 -req -in request.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out example.crt -days 5000
    # создаем публичный ключ из приватного
    openssl rsa -in private.key -pubout > public.key
     
Теперь необходимо настроить DNS.

PostmanQ должен представляться в команде HELO/EHLO своим полным доменным именем(FQDN) почты.

FQDN почты должно быть указано в A записи с внешним IP.

PTR запись должна указывать на FQDN почты.

MX запись должна указывать на FQDN почты.
 
Также необходимо указать DKIM и SPF записи.

    mail.example.com.                A           1.2.3.4
    4.3.2.1.in-addr.arpa.            IN PTR      mail.example.com. 
    _domainkey.example.com.          TXT         "t=s; o=~;"
    selector._domainkey.example.com. 3600 IN TXT "k=rsa\; t=s\; p=содержимое public.key" 
    example.com.                     IN TXT      "v=spf1 +a +mx ~all"
          
Selector может быть любым словом на латинице. Значение selector необходимо указать в настройках PostmanQ в поле dkimSelector.

Если PTR запись отсутствует, то письма могут попадать в спам, либо почтовые сервисы могут отклонять отправку.

Также необходимо увеличить количество открываемых файловых дескрипторов, иначе PostmanQ не сможет открывать новые соединения, и письма будут падать в одну из очередей для повторной отправки.

Затем устанавливаем AMQP-сервер, например [RabbitMQ](https://www.rabbitmq.com).
    
Теперь наши письма не будут попадать в спам, и все готово для установки PostmanQ.

## Установка

Сначала уcтанавливаем [go](http://golang.org/doc/install). Затем устанавливаем PostmanQ:

    cd /some/path && mkdir postmanq && cd postmanq/
    export GOPATH=/some/path/postmanq/
    export GOBIN=/some/path/postmanq/bin/
    go get -d github.com/boreevyuri/postmanq/cmd
    cd src/github.com/boreevyuri/postmanq
    git checkout v.3.1
    go install cmd/postmanq.go
    go install cmd/pmq-grep.go
    go install cmd/pmq-publish.go
    go install cmd/pmq-report.go
    ln -s /some/path/postmanq/bin/postmanq /usr/bin/
    ln -s /some/path/postmanq/bin/pmq-grep /usr/bin/
    ln -s /some/path/postmanq/bin/pmq-publish /usr/bin/
    ln -s /some/path/postmanq/bin/pmq-report /usr/bin/
    
Затем берем из репозитория config.yaml и пишем свой файл с настройками. Все настройки подробно описаны в самом config.yaml.

## Использование

    sudo rabbitmq-server -detached
    postmanq -f /path/to/config.yaml
    
## Утилиты

Для PostmanQ создано несколько утилит, призванных облегчить работу с логами и очередями рассылок - pmq-grep, pmq-publish, pmq-report.
Вызов каждой из утилит без аргументов покажет ее использование.

### pmq-grep

Если PostmanQ пишет логи в файл, то с помощью pmq-grep можно вытащить из лога все записи по определенному email получателя.

### pmq-publish

Если вы что то не прописали в DNS, или операционная система не может открыть столько соединений, сколько необходимо для PostmanQ, то велика вероятность, 
что письма не будут отправляться, и PostmanQ будет складывать письма в очередь для ошибок или в одну из очередей для повторной отправки.
После устранения проблемы, возможно, понадобится срочно разослать неотправленные письма. Как раз для этого и существует pmq-publish.
С помощью pmq-publish можно переложить письма, например, из очереди для ошибок в очередь для отправки, отфильтровав письма по коду ошибки, полученной от почтового сервиса.

### pmq-report

С помощью pmq-report можно посмотреть - по какой причине письмо попало в очередь для ошибок.

## Docker Качаем конфиг:
```bash
curl -o /path/to/config.yaml https://raw.githubusercontent.com/boreevyuri/postmanq/v.3.1/config.yaml
```
Настраиваем доступы к AMQP-серверу.

И запускаем, прокинув конфиг:

```bash
docker run -v `/path/to/config.yaml`:`/etc/postmanq.yaml` -d --restart unless-stopped --name postmanq boreevyuri/postmanq:latest  
```  