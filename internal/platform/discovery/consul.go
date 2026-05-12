package discovery

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
)

type Consul struct {
	client *api.Client
}

func NewConsul(address, datacenter string) (*Consul, error) {
	cfg := api.DefaultConfig()
	cfg.Address = address
	if datacenter != "" {
		cfg.Datacenter = datacenter
	}
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Consul{client: client}, nil
}

func (c *Consul) Register(serviceID, serviceName, serviceHost, servicePort, healthPath string) error {
	port, err := strconv.Atoi(servicePort)
	if err != nil {
		return fmt.Errorf("invalid service port %q: %w", servicePort, err)
	}
	checkURL := fmt.Sprintf("http://%s:%d%s", serviceHost, port, healthPath)
	reg := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: serviceHost,
		Port:    port,
		Check: &api.AgentServiceCheck{
			HTTP:                           checkURL,
			Method:                         "GET",
			Interval:                       "10s",
			Timeout:                        "3s",
			DeregisterCriticalServiceAfter: "1m",
		},
	}
	return c.client.Agent().ServiceRegister(reg)
}

func (c *Consul) Deregister(serviceID string) error {
	return c.client.Agent().ServiceDeregister(serviceID)
}

func (c *Consul) ResolveHealthy(serviceName string) (string, error) {
	entries, _, err := c.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no healthy instance for %s", serviceName)
	}
	pick := entries[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(entries))]
	return fmt.Sprintf("http://%s:%d", pick.Service.Address, pick.Service.Port), nil
}
