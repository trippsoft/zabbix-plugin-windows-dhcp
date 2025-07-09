package main

import (
	"fmt"
	"os/exec"

	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
	"golang.zabbix.com/sdk/plugin/container"
)

const (
	getScopesCmdlet = "Get-DhcpServerv4ScopeStatistics -ErrorAction SilentlyContinue | Select-Object -Property ScopeId, Free, InUse -ErrorAction SilentlyContinue | ConvertTo-Json -ErrorAction SilentlyContinue"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

type windowsDhcpPlugin struct {
	plugin.Base
}

var _ plugin.Exporter = (*windowsDhcpPlugin)(nil)

func run() error {
	p := &windowsDhcpPlugin{}

	plugin.RegisterMetrics(p, "WindowsDhcp", "windows_dhcp.scope.get", "The list of DHCP scopes.")

	h, err := container.NewHandler("WindowsDhcp")
	if err != nil {
		return errs.Wrap(err, "failed to create handler")
	}

	p.Logger = h

	err = h.Execute()
	if err != nil {
		return errs.Wrap(err, "failed to execute handler")
	}

	return nil
}

func (p *windowsDhcpPlugin) Export(key string, params []string, context plugin.ContextProvider) (any, error) {

	if key != "windows_dhcp.scope.get" {
		return nil, errs.Errorf("unknown key %q", key)
	}

	jsonResult, err := executePowershellCmdlet(getScopesCmdlet)

	if err != nil {
		return nil, errs.Wrap(err, "failed to execute PowerShell cmdlet to get DHCP scopes")
	}

	if len(jsonResult) == 0 {
		return "[]", nil
	}

	if jsonResult[0] != 91 { // Check if the result is a single object (e.g., "{...}")
		return fmt.Sprintf("[%s]", string(jsonResult)), nil
	}

	return string(jsonResult), nil
}

func executePowershellCmdlet(cmdlet string) ([]byte, error) {
	cmd := exec.Command("powershell.exe", "-nologo", "-noprofile", "-command", cmdlet)

	return cmd.CombinedOutput()
}
