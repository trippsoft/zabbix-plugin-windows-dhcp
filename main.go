package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"git.zabbix.com/ap/plugin-support/plugin"
	"git.zabbix.com/ap/plugin-support/plugin/container"
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

func run() error {
	p := &windowsDhcpPlugin{}

	plugin.RegisterMetrics(
		p,
		"WindowsDhcp",
		"scope_ids",
		"The list of DHCP scope IDs.",
		"scope_free",
		"The number of free IP addresses in the DHCP scope.",
		"scope_in_use",
		"The number of used IP addresses in the DHCP scope.",
	)

	h, err := container.NewHandler("WindowsDhcp")
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	p.Logger = &h

	err = h.Execute()
	if err != nil {
		return fmt.Errorf("failed to execute handler: %w", err)
	}

	return nil
}

func (p *windowsDhcpPlugin) Export(key string, params []string, context plugin.ContextProvider) (any, error) {
	switch key {
	case "scope_ids":
		return p.getScopeIDs()
	case "scope_free":
		return p.getScopeFree(params)
	case "scope_in_use":
		return p.getScopeInUse(params)
	default:
		return nil, fmt.Errorf("unknown item key %q", key)
	}
}

func (p *windowsDhcpPlugin) getScopeIDs() (any, error) {

	jsonResult, err := executePowershellCmdlet("Get-DhcpServerv4Scope | Select-Object -ExpandProperty ScopeId")

	if err != nil {
		return nil, fmt.Errorf("failed to execute PowerShell cmdlet to get DHCP scope IDs: %w", err)
	}

	result := make([]string, 0)

	if len(jsonResult) == 0 {
		return result, nil
	}

	err = json.Unmarshal(jsonResult, &result)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal scope IDs: %w", err)
	}

	return result, nil
}

func (p *windowsDhcpPlugin) getScopeFree(params []string) (any, error) {

	if len(params) == 0 {
		return nil, fmt.Errorf("scope ID is required for scope_free")
	}

	scopeID := params[0]
	cmdlet := fmt.Sprintf("Get-DhcpServerv4Scope -ScopeId %s -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Free -ErrorAction SilentlyContinue", scopeID)

	resultBytes, err := executePowershellCmdlet(cmdlet)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PowerShell cmdlet to get free IPs in scope %s: %w", scopeID, err)
	}

	if len(resultBytes) == 0 {
		return nil, fmt.Errorf("failed to retrieve free IPs in scope %s", scopeID)
	}

	result, err := strconv.Atoi(string(resultBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse free IPs in scope %s: %w", scopeID, err)
	}

	return result, nil
}

func (p *windowsDhcpPlugin) getScopeInUse(params []string) (any, error) {

	if len(params) == 0 {
		return nil, fmt.Errorf("scope ID is required for scope_free")
	}

	scopeID := params[0]
	cmdlet := fmt.Sprintf("Get-DhcpServerv4Scope -ScopeId %s -ErrorAction SilentlyContinue | Select-Object -ExpandProperty InUse -ErrorAction SilentlyContinue", scopeID)

	resultBytes, err := executePowershellCmdlet(cmdlet)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PowerShell cmdlet to get in-use IPs in scope %s: %w", scopeID, err)
	}

	if len(resultBytes) == 0 {
		return nil, fmt.Errorf("failed to retrieve in-use IPs in scope %s", scopeID)
	}

	result, err := strconv.Atoi(string(resultBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse in-use IPs in scope %s: %w", scopeID, err)
	}

	return result, nil
}

func executePowershellCmdlet(cmdlet string) ([]byte, error) {
	cmd := exec.Command(
		"powershell.exe",
		"-nologo",
		"-noprofile",
		"-command",
		fmt.Sprintf("{ %s }", cmdlet))

	return cmd.CombinedOutput()
}
