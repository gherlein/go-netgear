package models

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"github.com/gherlein/go-netgear/internal/common"
	"github.com/gherlein/go-netgear/internal/types"
)

type PoeCyclePowerCommand struct {
	Address string `required:"" help:"the Netgear switch's IP address or host name to connect to" short:"a"`
	Ports   []int  `required:"" help:"port number (starting with 1), use multiple times for cycling multiple ports at once" short:"p" name:"port"`
}

func (poe *PoeCyclePowerCommand) Run(args *types.GlobalOptions) error {
	model := args.Model
	if len(model) == 0 {
		var err error
		model, err = DetectNetgearModel(args, poe.Address)
		if err != nil {
			return err
		}
		args.Model = model

	}
	if common.IsModel30x(model) {
		return poe.cyclePowerGs30xEPx(args)
	}
	if common.IsModel316(model) {
		return poe.cyclePowerGs316EPx(args)
	}
	panic("model not supported")
}

func (poe *PoeCyclePowerCommand) cyclePowerGs30xEPx(args *types.GlobalOptions) error {
	poeExt := &PoeExt{}

	settings, err := requestPoeConfiguration(args, poe.Address, poeExt)
	if err != nil {
		return err
	}

	poeSettings := url.Values{
		"hash":   {poeExt.Hash},
		"ACTION": {"Reset"},
	}

	for _, switchPort := range poe.Ports {
		if switchPort < 1 || switchPort > len(settings) {
			return errors.New(fmt.Sprintf("given port id %d, doesn't fit in range 1..%d", switchPort, len(settings)))
		}
		poeSettings.Add(fmt.Sprintf("port%d", switchPort-1), "checked")
	}

	result, err := requestPoeSettingsUpdate(args, poe.Address, poeSettings.Encode())
	if err != nil {
		return err
	}
	if result != "SUCCESS" {
		return errors.New(result)
	}

	statuses, err := requestPoeStatus(args, poe.Address)
	if err != nil {
		return err
	}
	statuses = common.Filter(statuses, func(status PoePortStatus) bool {
		return slices.Contains(poe.Ports, int(status.PortIndex))
	})

	return nil
}

func (poe *PoeCyclePowerCommand) cyclePowerGs316EPx(args *types.GlobalOptions) error {
	for _, switchPort := range poe.Ports {
		if switchPort < 1 || switchPort > gs316NoPoePorts {
			return errors.New(fmt.Sprintf("given port id %d, doesn't fit in range 1..%d", switchPort, gs316NoPoePorts))
		}
	}

	_, token, err := common.ReadTokenAndModel2GlobalOptions(args, poe.Address)
	if err != nil {
		return err
	}
	urlStr := fmt.Sprintf("http://%s/iss/specific/poePortConf.html", poe.Address)
	reqForm := url.Values{}
	reqForm.Add("Gambit", token)
	reqForm.Add("TYPE", "resetPoe")
	reqForm.Add("PoePort", createPortResetPayloadGs316EPx(poe.Ports))
	result, err := common.DoHttpRequestAndReadResponse(args, http.MethodPost, poe.Address, urlStr, reqForm.Encode())
	if err != nil {
		return err
	}
	if args.Verbose {
		fmt.Println(result)
	}
	if result != "SUCCESS" {
		return errors.New(result)
	}

	statuses, err := requestPoeStatus(args, poe.Address)
	if err != nil {
		return err
	}
	statuses = common.Filter(statuses, func(status PoePortStatus) bool {
		return slices.Contains(poe.Ports, int(status.PortIndex))
	})
	prettyPrintPoePortStatus(args.OutputFormat, statuses)
	return nil
}

func createPortResetPayloadGs316EPx(poePorts []int) string {
	result := strings.Builder{}
	for i := 0; i < gs316NoPoePorts; i++ {
		written := false
		for _, p := range poePorts {
			if p-1 == i {
				result.WriteString("1")
				written = true
				break
			}
		}
		if !written {
			result.WriteString("0")
		}
	}
	return result.String()
}
