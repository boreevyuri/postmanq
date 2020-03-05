package grep

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/boreevyuri/postmanq/common"
	yaml "gopkg.in/yaml.v2"
)

var (
	// сервис ищущий сообщения в логе об отправке письма
	service *Service

	// регулярное выражение, по которому находим начало отправки
	mailIDRegex = regexp.MustCompile(`mail#((\d)+)+`)
)

// Service сервис ищущий сообщения в логе об отправке письма
type Service struct {
	// путь до файла с логами
	Output string `yaml:"logOutput"`

	// файл с логами
	logFile *os.File
}

// Inst создает новый сервис поиска по логам
func Inst() common.GrepService {
	if service == nil {
		service = new(Service)
	}
	return service
}

// OnInit инициализирует сервис
func (s *Service) OnInit(event *common.ApplicationEvent) {
	var err error
	err = yaml.Unmarshal(event.Data, s)
	if err == nil {
		if common.FilenameRegex.MatchString(s.Output) {
			s.logFile, err = os.OpenFile(s.Output, os.O_RDONLY, os.ModePerm)
			if err != nil {
				fmt.Println(err)
				common.App.Events() <- common.NewApplicationEvent(common.FinishApplicationEventKind)
			}
		} else {
			fmt.Println("logOutput should be a file")
			common.App.Events() <- common.NewApplicationEvent(common.FinishApplicationEventKind)
		}
	} else {
		fmt.Println("service can't unmarshal config file")
		common.App.Events() <- common.NewApplicationEvent(common.FinishApplicationEventKind)
	}
}

// OnGrep ищет логи об отправке письма
func (s *Service) OnGrep(event *common.ApplicationEvent) {
	scanner := bufio.NewScanner(s.logFile)
	scanner.Split(bufio.ScanLines)
	lines := make(chan string)
	outs := make(chan string)

	go func() {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	var expr string
	hasEnvelope := event.GetStringArg("envelope") == ""
	if hasEnvelope {
		expr = fmt.Sprintf("envelope - %s, recipient - %s to mailer", event.GetStringArg("envelope"), event.GetStringArg("recipient"))
	} else {
		expr = fmt.Sprintf("recipient - %s to mailer", event.GetStringArg("recipient"))
	}

	go func() {
		var successExpr, failExpr, failPubExpr, delayExpr, limitExpr string
		var mailID string
		for line := range lines {
			if mailID == "" {
				if strings.Contains(line, expr) {
					results := mailIDRegex.FindStringSubmatch(line)
					if len(results) == 3 {
						mailID = results[1]

						successExpr = fmt.Sprintf("%s success send", mailID)
						failExpr = fmt.Sprintf("%s publish failed mail to queue", mailID)
						failPubExpr = fmt.Sprintf("%s can't publish failed mail to queue", mailID)
						delayExpr = fmt.Sprintf("%s detect old dlx queue", mailID)
						limitExpr = fmt.Sprintf("%s detect overlimit", mailID)

						outs <- line
					}
				}
			} else {
				if strings.Contains(line, mailID) {
					outs <- line
				}
				if strings.Contains(line, successExpr) ||
					strings.Contains(line, failExpr) ||
					strings.Contains(line, failPubExpr) ||
					strings.Contains(line, delayExpr) ||
					strings.Contains(line, limitExpr) {
					mailID = ""
				}
			}
		}
		close(lines)
	}()

	for out := range outs {
		fmt.Println(out)
	}
	close(outs)
	common.App.Events() <- common.NewApplicationEvent(common.FinishApplicationEventKind)
}

// выводит логи в терминал
func (s *Service) print(mailID string, lines []string, wg *sync.WaitGroup) {
	out := new(bytes.Buffer)

	for _, line := range lines {
		if strings.Contains(line, mailID) {
			out.WriteString(line)
			out.WriteString("\n")
		}
	}

	fmt.Println(out.String())
	wg.Done()
}

// OnFinish завершает работу сервиса
func (s *Service) OnFinish(event *common.ApplicationEvent) {
	s.logFile.Close()
}
