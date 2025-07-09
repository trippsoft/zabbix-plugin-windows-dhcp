package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
	"golang.zabbix.com/sdk/plugin/container"
)

const (
	getScopeIdsCmdlet = "Get-DhcpServerv4Scope -ErrorAction SilentlyContinue | Select-Object -ExpandProperty ScopeId -ErrorAction SilentlyContinue | ConvertTo-Json -ErrorAction SilentlyContinue"
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

	plugin.RegisterMetrics(
		p,
		"WindowsDhcp",
		"windows_dhcp.scope_ids",
		"The list of DHCP scope IDs.",
		"windows_dhcp.scope_free",
		"The number of free IP addresses in the DHCP scope.",
		"windows_dhcp.scope_in_use",
		"The number of used IP addresses in the DHCP scope.",
	)

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
	switch key {
	case "windows_dhcp.scope_ids":
		return p.getScopeIDs()
	case "windows_dhcp.scope_free":
		return p.getScopeFree(params)
	case "windows_dhcp.scope_in_use":
		return p.getScopeInUse(params)
	default:
		return nil, errs.Errorf("unknown item key %q", key)
	}
}

func (p *windowsDhcpPlugin) getScopeIDs() (any, error) {

	jsonResult, err := executePowershellCmdlet("Get-DhcpServerv4Scope | Select-Object -ExpandProperty ScopeId")

	if err != nil {
		return nil, errs.Wrap(err, "failed to execute PowerShell cmdlet to get DHCP scope IDs")
	}

	result := make([]string, 0)

	if len(jsonResult) == 0 {
		return result, nil
	}

	if jsonResult[0] == 34 { // Check if the result is a single string (e.g., "<string>")

		singleResult := string(jsonResult)
		singleResult = strings.Trim(singleResult, "\"\r\n") // Remove surrounding quotes and newlines
		singleResult = strings.TrimSpace(singleResult)      // Remove any leading/trailing whitespace

		result = append(result, singleResult)

		return result, nil
	}

	err = json.Unmarshal(jsonResult, &result)

	if err != nil {
		return nil, errs.Wrapf(err, "failed to unmarshal scope IDs from %s", jsonResult)
	}

	return result, nil
}

func (p *windowsDhcpPlugin) getScopeFree(params []string) (any, error) {

	if len(params) == 0 {
		return nil, errs.Errorf("scope ID is required for scope_free")
	}

	scopeID := params[0]
	cmdlet := fmt.Sprintf("Get-DhcpServerv4Scope -ScopeId %s -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Free -ErrorAction SilentlyContinue", scopeID)

	resultBytes, err := executePowershellCmdlet(cmdlet)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to execute PowerShell cmdlet to get free IPs in scope %s", scopeID)
	}

	if len(resultBytes) == 0 {
		return nil, errs.Wrapf(err, "failed to retrieve free IPs in scope %s", scopeID)
	}

	result, err := strconv.Atoi(string(resultBytes))
	if err != nil {
		return nil, errs.Wrapf(err, "failed to parse free IPs in scope %s", scopeID)
	}

	return result, nil
}

func (p *windowsDhcpPlugin) getScopeInUse(params []string) (any, error) {

	if len(params) == 0 {
		return nil, errs.Errorf("scope ID is required for scope_free")
	}

	scopeID := params[0]
	cmdlet := fmt.Sprintf("Get-DhcpServerv4Scope -ScopeId %s -ErrorAction SilentlyContinue | Select-Object -ExpandProperty InUse -ErrorAction SilentlyContinue", scopeID)

	resultBytes, err := executePowershellCmdlet(cmdlet)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to execute PowerShell cmdlet to get in-use IPs in scope %s", scopeID)
	}

	if len(resultBytes) == 0 {
		return nil, errs.Wrapf(err, "failed to retrieve in-use IPs in scope %s", scopeID)
	}

	result, err := strconv.Atoi(string(resultBytes))
	if err != nil {
		return nil, errs.Wrapf(err, "failed to parse in-use IPs in scope %s", scopeID)
	}

	return result, nil
}

func executePowershellCmdlet(cmdlet string) ([]byte, error) {
	cmd := exec.Command(
		"powershell.exe",
		"-nologo",
		"-noprofile",
		"-command",
		cmdlet)

	return cmd.CombinedOutput()
}
