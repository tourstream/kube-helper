package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"net/http"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func (a *applicationService) addNewRulesToLoadBalancer(addresses string) error {
	addressList := a.splitIntoAddresses(addresses)

	for _, address := range addressList {
		newRule, err := a.createNewRule(address)

		if err != nil {
			fmt.Fprintln(writer, err.Error())
			return err
		}

		_, err = a.computeService.GlobalForwardingRules.Insert(projectId, newRule).Do()

		if err != nil {
			return err
		}

		fmt.Fprintf(writer, "Adress %s added\n", address)
	}

	return nil
}

func (a *applicationService) splitIntoAddresses(addresses string) []NamedAddress {
	var addressList []NamedAddress
	for _, address := range strings.Split(addresses, ",") {
		valid, namedAddress := a.getValidNamedAddress(address)
		if valid {
			addressList = append(addressList, namedAddress)
		}
	}

	return addressList
}

func (a *applicationService) getValidNamedAddress(address string) (bool, NamedAddress) {
	if address == "" {
		fmt.Fprintf(writer, "Empty Address\n")
		return false, NamedAddress{}
	}

	parts := strings.Split(address, ":")

	if len(parts) != 2 || parts[1] != "80" && parts[1] != "443" {
		fmt.Fprintf(writer, "Invalid Address \"%s\"\n", address)
		return false, NamedAddress{}
	}

	return true, NamedAddress{ipName: parts[0], port: parts[1]}
}

func (a *applicationService) createNewRule(address NamedAddress) (*compute.ForwardingRule, error) {
	rule, err := a.getRuleToCopy(address, 60)

	if err != nil {
		return nil, err
	}

	ipAddress, err := a.getIpAddressForNewRule(address.ipName)

	if err != nil {
		return nil, err
	}

	newName := ipAddress.Name + "-fr-" + address.port

	err = a.newRulePreCheck(newName)

	if err != nil {
		return nil, err
	}

	newRule := new(compute.ForwardingRule)
	newRule.Target = rule.Target
	newRule.IPAddress = ipAddress.Address
	newRule.IPProtocol = rule.IPProtocol
	newRule.LoadBalancingScheme = rule.LoadBalancingScheme
	newRule.Name = newName
	newRule.PortRange = address.port + "-" + address.port

	return newRule, nil
}

func (a *applicationService) newRulePreCheck(newName string) error {
	_, err := a.computeService.GlobalForwardingRules.Get(projectId, newName).Do()

	if err == nil {
		return errors.New("Rule " + newName + " already exists")
	}
	if err.(*googleapi.Error).Code != http.StatusNotFound {
		return err
	}

	return nil
}

func (a *applicationService) getRuleToCopy(address NamedAddress, retries int) (*compute.ForwardingRule, error) {
	for i := 0; i < retries; i++ {
		rule, err := a.findRule(address)

		if err != nil {
			return nil, err
		}

		if rule != nil {
			return rule, nil
		}
	}

	return nil, errors.New("No existing rule for namespace found")
}

func (a *applicationService) findRule(address NamedAddress) (*compute.ForwardingRule, error) {
	rules, err := a.computeService.GlobalForwardingRules.List(projectId).Do()

	if err != nil {
		return nil, err
	}

	for _, rule := range rules.Items {
		if strings.Contains(rule.Name, kb8Namespace) && strings.Contains(rule.PortRange, address.port) {
			return rule, nil
		}
	}

	fmt.Fprintln(writer, "Waiting for first Rule to be appended")
	clock.Sleep(time.Second * 5)

	return nil, nil
}

func (a *applicationService) getIpAddressForNewRule(address string) (*compute.Address, error) {
	ipAddress, err := a.getIpAddressByName(address)

	if err != nil {
		return nil, err
	}

	if ipAddress.Status == "IN USE" {
		return nil, errors.New("Ip Address already used")
	}

	return ipAddress, nil
}

func (a *applicationService) getIpAddressByName(address string) (*compute.Address, error) {
	ips, err := a.computeService.GlobalAddresses.List(projectId).Do()

	if err != nil {
		return nil, err
	}

	for _, ip := range ips.Items {
		if strings.Contains(ip.Name, address) {
			return ip, nil
		}
	}

	return nil, errors.New("Ip Address not found")
}
