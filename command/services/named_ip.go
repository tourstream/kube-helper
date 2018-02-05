package services

import (
	"github.com/urfave/cli"
	"kube-helper/service"
	"google.golang.org/api/compute/v1"
	"strings"
)

func CmdAddRuleToLoadBalancer(c *cli.Context) error {
	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	projectId = configContainer.Cluster.ProjectID
	if projectId == "" {
		return cli.NewExitError("projectId is empty", 1)
	}

	kubernetesNamespace := configContainer.Namespace.Prefix + "-" + c.Args().Get(0)
	address := c.String("address")

	if address == "" {
		return cli.NewExitError("address not set", 1)
	}

	builder := new(service.Builder)
	computeService, err := builder.GetComputeService()

	rules, err := computeService.GlobalForwardingRules.List(projectId).Do()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	for _, rule := range rules.Items {
		if strings.Contains(rule.Name, kubernetesNamespace) {

			err = addRuleToLoadBalancerWithIp(rule, computeService, address)

			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
		}

	}

	return nil
}

func getIpAddressByName(computeService *compute.Service, address string) (*compute.Address, error) {
	ips, err := computeService.GlobalAddresses.List(projectId).Do()

	if err != nil {
		return nil, cli.NewExitError(err.Error(), 1)
	}

	for _, ip := range ips.Items {
		if strings.Contains(ip.Name, address) {
			return ip, nil
		}
	}

	return nil, cli.NewExitError("Ip Address not found", 1);
}

func addRuleToLoadBalancerWithIp(rule *compute.ForwardingRule, computeService *compute.Service, address string) error {
	ipAddress, err := getIpAddressByName(computeService, address)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if ipAddress.Status == "IN USE" {
		return cli.NewExitError("Ip Address allready used", 1)
	}

	newName := ipAddress.Name + "-fr-" + rule.PortRange
	_, err = computeService.GlobalForwardingRules.Get(projectId, newName).Do()

	if err == nil {
		return cli.NewExitError("Rule already exists", 1)
	}

	if !strings.Contains(err.Error(), "Error 404") {
		return cli.NewExitError(err.Error(), 1)
	}

	newRule := new(compute.ForwardingRule)
	newRule.Target = rule.Target
	newRule.IPAddress = ipAddress.Address
	newRule.IPProtocol = rule.IPProtocol
	newRule.LoadBalancingScheme = rule.LoadBalancingScheme
	newRule.Name = newName
	newRule.PortRange = rule.PortRange

	_, err = computeService.GlobalForwardingRules.Insert(projectId, newRule).Do()

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

func CmdRemoveRuleFromLoadBalancer(c *cli.Context) error {
	configContainer, err := configLoader.LoadConfigFromPath(c.String("config"))

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	projectId = configContainer.Cluster.ProjectID
	if projectId == "" {
		return cli.NewExitError("projectId is empty", 1)
	}

	address := c.String("address")
	if address == "" {
		return cli.NewExitError("address not set", 1)
	}

	builder := new(service.Builder)
	computeService, err := builder.GetComputeService()

	ipAddress, err := getIpAddressByName(computeService, address)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	for _, rule := range ipAddress.Users {
		_, err = computeService.GlobalForwardingRules.Delete(projectId, getResourceName(rule)).Do()

		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
	}

	return nil
}
