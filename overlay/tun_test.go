package overlay

import (
	"fmt"
	"net"
	"testing"

	"github.com/slackhq/nebula/config"
	"github.com/slackhq/nebula/test"
	"github.com/stretchr/testify/assert"
)

func Test_ParseRoutes(t *testing.T) {
	l := test.NewLogger()
	c := config.NewC(l)
	_, n, _ := net.ParseCIDR("10.0.0.0/24")

	// test no routes config
	routes, err := ParseRoutes(c, n)
	assert.Nil(t, err)
	assert.Len(t, routes, 0)

	// not an array
	c.Settings["tun"] = map[interface{}]interface{}{"routes": "hi"}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "tun.routes is not an array")

	// no routes
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, err)
	assert.Len(t, routes, 0)

	// weird route
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{"asdf"}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1 in tun.routes is invalid")

	// no mtu
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{map[interface{}]interface{}{}}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.mtu in tun.routes is not present")

	// bad mtu
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{map[interface{}]interface{}{"mtu": "nope"}}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.mtu in tun.routes is not an integer: strconv.Atoi: parsing \"nope\": invalid syntax")

	// low mtu
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{map[interface{}]interface{}{"mtu": "499"}}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.mtu in tun.routes is below 500: 499")

	// missing route
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{map[interface{}]interface{}{"mtu": "500"}}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.route in tun.routes is not present")

	// unparsable route
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{map[interface{}]interface{}{"mtu": "500", "route": "nope"}}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.route in tun.routes failed to parse: invalid CIDR address: nope")

	// below network range
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{map[interface{}]interface{}{"mtu": "500", "route": "1.0.0.0/8"}}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.route in tun.routes is not contained within the network attached to the certificate; route: 1.0.0.0/8, network: 10.0.0.0/24")

	// above network range
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{map[interface{}]interface{}{"mtu": "500", "route": "10.0.1.0/24"}}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.route in tun.routes is not contained within the network attached to the certificate; route: 10.0.1.0/24, network: 10.0.0.0/24")

	// happy case
	c.Settings["tun"] = map[interface{}]interface{}{"routes": []interface{}{
		map[interface{}]interface{}{"mtu": "9000", "route": "10.0.0.0/29"},
		map[interface{}]interface{}{"mtu": "8000", "route": "10.0.0.1/32"},
	}}
	routes, err = ParseRoutes(c, n)
	assert.Nil(t, err)
	assert.Len(t, routes, 2)

	tested := 0
	for _, r := range routes {
		if r.MTU == 8000 {
			assert.Equal(t, "10.0.0.1/32", r.Cidr.String())
			tested++
		} else {
			assert.Equal(t, 9000, r.MTU)
			assert.Equal(t, "10.0.0.0/29", r.Cidr.String())
			tested++
		}
	}

	if tested != 2 {
		t.Fatal("Did not see both routes")
	}
}

func Test_ParseUnsafeRoutes(t *testing.T) {
	l := test.NewLogger()
	c := config.NewC(l)
	_, n, _ := net.ParseCIDR("10.0.0.0/24")

	// test no routes config
	routes, err := ParseUnsafeRoutes(c, n)
	assert.Nil(t, err)
	assert.Len(t, routes, 0)

	// not an array
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": "hi"}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "tun.unsafe_routes is not an array")

	// no routes
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, err)
	assert.Len(t, routes, 0)

	// weird route
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{"asdf"}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1 in tun.unsafe_routes is invalid")

	// no via
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.via in tun.unsafe_routes is not present")

	// invalid via
	for _, invalidValue := range []interface{}{
		127, false, nil, 1.0, []string{"1", "2"},
	} {
		c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"via": invalidValue}}}
		routes, err = ParseUnsafeRoutes(c, n)
		assert.Nil(t, routes)
		assert.EqualError(t, err, fmt.Sprintf("entry 1.via in tun.unsafe_routes is not a string: found %T", invalidValue))
	}

	// unparsable via
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"mtu": "500", "via": "nope"}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.via in tun.unsafe_routes failed to parse address: nope")

	// missing route
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"via": "127.0.0.1", "mtu": "500"}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.route in tun.unsafe_routes is not present")

	// unparsable route
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"via": "127.0.0.1", "mtu": "500", "route": "nope"}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.route in tun.unsafe_routes failed to parse: invalid CIDR address: nope")

	// within network range
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"via": "127.0.0.1", "route": "10.0.0.0/24"}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.route in tun.unsafe_routes is contained within the network attached to the certificate; route: 10.0.0.0/24, network: 10.0.0.0/24")

	// below network range
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"via": "127.0.0.1", "route": "1.0.0.0/8"}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Len(t, routes, 1)
	assert.Nil(t, err)

	// above network range
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"via": "127.0.0.1", "route": "10.0.1.0/24"}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Len(t, routes, 1)
	assert.Nil(t, err)

	// no mtu
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"via": "127.0.0.1", "route": "1.0.0.0/8"}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Len(t, routes, 1)
	assert.Equal(t, DefaultMTU, routes[0].MTU)

	// bad mtu
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"via": "127.0.0.1", "mtu": "nope"}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.mtu in tun.unsafe_routes is not an integer: strconv.Atoi: parsing \"nope\": invalid syntax")

	// low mtu
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{map[interface{}]interface{}{"via": "127.0.0.1", "mtu": "499"}}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, routes)
	assert.EqualError(t, err, "entry 1.mtu in tun.unsafe_routes is below 500: 499")

	// happy case
	c.Settings["tun"] = map[interface{}]interface{}{"unsafe_routes": []interface{}{
		map[interface{}]interface{}{"via": "127.0.0.1", "mtu": "9000", "route": "1.0.0.0/29"},
		map[interface{}]interface{}{"via": "127.0.0.1", "mtu": "8000", "route": "1.0.0.1/32"},
		map[interface{}]interface{}{"via": "127.0.0.1", "mtu": "1500", "metric": 1234, "route": "1.0.0.2/32"},
	}}
	routes, err = ParseUnsafeRoutes(c, n)
	assert.Nil(t, err)
	assert.Len(t, routes, 3)

	tested := 0
	for _, r := range routes {
		if r.MTU == 8000 {
			assert.Equal(t, "1.0.0.1/32", r.Cidr.String())
			tested++
		} else if r.MTU == 9000 {
			assert.Equal(t, 9000, r.MTU)
			assert.Equal(t, "1.0.0.0/29", r.Cidr.String())
			tested++
		} else {
			assert.Equal(t, 1500, r.MTU)
			assert.Equal(t, 1234, r.Metric)
			assert.Equal(t, "1.0.0.2/32", r.Cidr.String())
			tested++
		}
	}

	if tested != 3 {
		t.Fatal("Did not see both unsafe_routes")
	}
}
