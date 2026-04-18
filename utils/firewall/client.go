package firewall

import (
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/firewall/client"
)

type FirewallClient interface {
	Name() string // ufw firewalld
	Start() error
	Stop() error
	Restart() error
	Reload() error
	Status() (string, error) // running not running
	Version() (string, error)

	ListPort() ([]client.FireInfo, error)
	ListForward() ([]client.FireInfo, error)
	ListAddress() ([]client.FireInfo, error)

	Port(port client.FireInfo, operation string) error
	RichRules(rule client.FireInfo, operation string) error
	PortForward(info client.Forward, operation string) error

	EnableForward() error
}

func NewFirewallClient() (FirewallClient, error) {
	firewalld := cmd.Which("firewalld")
	ufw := cmd.Which("ufw")

	if firewalld && ufw {
		return nil, buserr.New(constant.ErrFirewallBoth)
	}

	if firewalld {
		return client.NewFirewalld()
	}
	if ufw {
		return client.NewUfw()
	}
	return nil, buserr.New(constant.ErrFirewallNone)
}
