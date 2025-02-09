package overlay

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"

	"github.com/songgao/water"
)

type WindowsWaterTun struct {
	Device       string
	Cidr         *net.IPNet
	MTU          int
	UnsafeRoutes []Route

	*water.Interface
}

func newWindowsWaterTun(deviceName string, cidr *net.IPNet, defaultMTU int, unsafeRoutes []Route, txQueueLen int) (ifce *WindowsWaterTun, err error) {
	// NOTE: You cannot set the deviceName under Windows, so you must check tun.Device after calling .Activate()
	return &WindowsWaterTun{
		Cidr:         cidr,
		MTU:          defaultMTU,
		UnsafeRoutes: unsafeRoutes,
	}, nil
}

func (c *WindowsWaterTun) Activate() error {
	var err error
	c.Interface, err = water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			ComponentID: "tap0901",
			Network:     c.Cidr.String(),
		},
	})
	if err != nil {
		return fmt.Errorf("Activate failed: %v", err)
	}

	c.Device = c.Interface.Name()

	// TODO use syscalls instead of exec.Command
	err = exec.Command(
		`C:\Windows\System32\netsh.exe`, "interface", "ipv4", "set", "address",
		fmt.Sprintf("name=%s", c.Device),
		"source=static",
		fmt.Sprintf("addr=%s", c.Cidr.IP),
		fmt.Sprintf("mask=%s", net.IP(c.Cidr.Mask)),
		"gateway=none",
	).Run()
	if err != nil {
		return fmt.Errorf("failed to run 'netsh' to set address: %s", err)
	}
	err = exec.Command(
		`C:\Windows\System32\netsh.exe`, "interface", "ipv4", "set", "interface",
		c.Device,
		fmt.Sprintf("mtu=%d", c.MTU),
	).Run()
	if err != nil {
		return fmt.Errorf("failed to run 'netsh' to set MTU: %s", err)
	}

	iface, err := net.InterfaceByName(c.Device)
	if err != nil {
		return fmt.Errorf("failed to find interface named %s: %v", c.Device, err)
	}

	for _, r := range c.UnsafeRoutes {
		err = exec.Command(
			"C:\\Windows\\System32\\route.exe", "add", r.Cidr.String(), r.Via.String(), "IF", strconv.Itoa(iface.Index), "METRIC", strconv.Itoa(r.Metric),
		).Run()
		if err != nil {
			return fmt.Errorf("failed to add the unsafe_route %s: %v", r.Cidr.String(), err)
		}
	}

	return nil
}

func (c *WindowsWaterTun) CidrNet() *net.IPNet {
	return c.Cidr
}

func (c *WindowsWaterTun) DeviceName() string {
	return c.Device
}

func (c *WindowsWaterTun) WriteRaw(b []byte) error {
	_, err := c.Write(b)
	return err
}

func (c *WindowsWaterTun) Close() error {
	if c.Interface == nil {
		return nil
	}

	return c.Interface.Close()
}

func (t *WindowsWaterTun) NewMultiQueueReader() (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("TODO: multiqueue not implemented for windows")
}
