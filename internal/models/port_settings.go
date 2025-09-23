package models

import (
	"github.com/gherlein/go-netgear/internal/formatter"
	"github.com/gherlein/go-netgear/internal/common"
	"github.com/gherlein/go-netgear/internal/types"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"strconv"
	"strings"
)

type PortCommand struct {
	PortSettingsCommand PortSettingsCommand `cmd:"" name:"settings" help:"show switch port settings" default:"1"`
	PortSetCommand      PortSetCommand      `cmd:"" name:"set" help:"set properties for a port number"`
}

type PortSettingsCommand struct {
	Address string `required:"" help:"the Netgear switch's IP address or host name to connect to" short:"a"`
}

func (port *PortSettingsCommand) Run(args *types.GlobalOptions) error {
	settings, _, err := requestPortSettings(args, port.Address)
	if err != nil {
		return err
	}
	prettyPrintPortSettings(args.Model, args.OutputFormat, settings)
	return nil
}

func requestPortSettings(args *types.GlobalOptions, host string) (portSettings []PortSetting, hash string, err error) {
	model, _, err := common.ReadTokenAndModel2GlobalOptions(args, host)
	if err != nil {
		return portSettings, hash, err
	}

	var requestUrl string
	if common.IsModel30x(model) {
		requestUrl = fmt.Sprintf("http://%s/dashboard.cgi", host)
	} else if common.IsModel316(model) {
		requestUrl = fmt.Sprintf("http://%s/iss/specific/dashboard.html", host)
	} else {
		panic("model not supported")
	}

	dashboardData, err := common.RequestPage(args, host, requestUrl)
	if err != nil {
		return portSettings, hash, err
	}

	if common.CheckIsLoginRequired(dashboardData) {
		return portSettings, hash, errors.New("no content. please, (re-)login first")
	}

	hash, err = findHashInHtml(model, strings.NewReader(dashboardData))
	if err != nil {
		return portSettings, hash, err
	}

	portSettings, err = findPortSettingsInHtml(model, strings.NewReader(dashboardData))

	if err != nil {
		return portSettings, hash, err
	}

	return portSettings, hash, err
}

func prettyPrintPortSettings(model types.NetgearModel, format formatter.OutputFormat, settings []PortSetting) {

	var header = []string{"Port ID", "Port Name", "Speed", "Ingress Limit", "Egress Limit", "Flow Control", "Port Status", "Link Speed"}
	var content [][]string

	for _, setting := range settings {
		var row []string
		row = append(row, fmt.Sprintf("%d", setting.Index))
		row = append(row, setting.Name)
		if common.IsModel30x(model) {
			setting.Speed = bidiMapLookup(setting.Speed, portSpeedMap)
		}
		row = append(row, setting.Speed)
		if common.IsModel30x(model) {
			setting.IngressRateLimit = bidiMapLookup(setting.IngressRateLimit, portRateLimitMap)
		}
		row = append(row, setting.IngressRateLimit)
		if common.IsModel30x(model) {
			setting.EgressRateLimit = bidiMapLookup(setting.EgressRateLimit, portRateLimitMap)
		}
		row = append(row, setting.EgressRateLimit)
		if common.IsModel30x(model) {
			setting.FlowControl = bidiMapLookup(setting.FlowControl, portFlowControlMap)
		}
		row = append(row, setting.FlowControl)
		row = append(row, setting.PortStatus)
		row = append(row, setting.LinkSpeed)
		content = append(content, row)
	}
	switch format {
	case formatter.MarkdownFormat:
		formatter.PrintMarkdownTable(header, content)
	case formatter.JsonFormat:
		formatter.PrintJsonDataTable("port_settings", header, content)
	default:
		panic("not implemented format: " + format)
	}

}

func findPortSettingsInHtml(model types.NetgearModel, reader io.Reader) ([]PortSetting, error) {
	if common.IsModel30x(model) {
		return findPortSettingsInGs30xEPxHtml(reader)
	}
	if common.IsModel316(model) {
		return findPortSettingsInGs316EPxHtml(reader)
	}
	panic("model not supported")
}

func findPortSettingsInGs30xEPxHtml(reader io.Reader) (ports []PortSetting, err error) {

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return ports, err
	}

	doc.Find("li.list_item").Each(func(i int, s *goquery.Selection) {
		portCfg := PortSetting{}

		id, _ := s.Find("input[type=hidden].port").Attr("value")
		var id64, _ = strconv.ParseInt(id, 10, 8)
		portCfg.Index = int8(id64)
		portCfg.Name, _ = s.Find("input[type=hidden].portName").Attr("value")
		portCfg.Speed, _ = s.Find("input[type=hidden].Speed").Attr("value")
		portCfg.IngressRateLimit, _ = s.Find("input[type=hidden].ingressRate").Attr("value")
		portCfg.EgressRateLimit, _ = s.Find("input[type=hidden].egressRate").Attr("value")
		portCfg.FlowControl, _ = s.Find("input[type=hidden].flowCtr").Attr("value")
		portCfg.LinkSpeed, _ = s.Find("input[type=hidden].LinkedSpeed").Attr("value")
		portCfg.PortStatus = strings.TrimSpace(s.Find("span.pull-right").Text())
		ports = append(ports, portCfg)
	})

	return ports, nil
}

func findPortSettingsInGs316EPxHtml(reader io.Reader) (ports []PortSetting, err error) {

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return ports, err
	}

	doc.Find("div.dashboard-port-status").Each(func(i int, s *goquery.Selection) {
		s.Find("span.port-number").Each(func(i int, selection *goquery.Selection) {
			ports = append(ports, PortSetting{})
		})

		s.Find("span.port-number").Each(func(i int, selection *goquery.Selection) {
			var id64, _ = strconv.ParseInt(strings.TrimSpace(selection.Text()), 10, 8)
			ports[i].Index = int8(id64)
		})
		s.Find("span.port-name span.name").Each(func(i int, selection *goquery.Selection) {
			ports[i].Name = strings.TrimSpace(selection.Text())
		})
		s.Find("p.speed-text").Each(func(i int, selection *goquery.Selection) {
			ports[i].Speed = strings.TrimSpace(selection.Text())
		})
		s.Find("p.ingress-text").Each(func(i int, selection *goquery.Selection) {
			ports[i].IngressRateLimit = strings.TrimSpace(selection.Text())
		})
		s.Find("p.egress-text").Each(func(i int, selection *goquery.Selection) {
			ports[i].EgressRateLimit = strings.TrimSpace(selection.Text())
		})
		s.Find("p.flow-text").Each(func(i int, selection *goquery.Selection) {
			ports[i].FlowControl = strings.TrimSpace(selection.Text())
		})
		s.Find("span.status-on-port").Each(func(i int, selection *goquery.Selection) {
			ports[i].PortStatus = strings.TrimSpace(selection.Text())
		})
		s.Find("p.link-speed-text").Each(func(i int, selection *goquery.Selection) {
			ports[i].LinkSpeed = strings.TrimSpace(selection.Text())
		})
	})

	return ports, nil
}
