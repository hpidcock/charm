package charm_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"gopkg.in/juju/charm.v2"
	gc "launchpad.net/gocheck"
)

type bundleDataSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&bundleDataSuite{})

const mediawikiBundle = `
series: precise
services: 
    mediawiki: 
        charm: "cs:precise/mediawiki-10"
        num_units: 1
        options: 
            debug: false
            name: Please set name of wiki
            skin: vector
        annotations: 
            "gui-x": 609
            "gui-y": -15
    mysql: 
        charm: "cs:precise/mysql-28"
        num_units: 2
        to: [0, mediawiki/0]
        options: 
            "binlog-format": MIXED
            "block-size": 5
            "dataset-size": "80%"
            flavor: distro
            "ha-bindiface": eth0
            "ha-mcastport": 5411
        annotations: 
            "gui-x": 610
            "gui-y": 255
        constraints: "mem=8g"
relations:
    - ["mediawiki:db", "mysql:db"]
    - ["mysql:foo", "mediawiki:bar"]
machines:
    0:
         constraints: 'arch=amd64 mem=4g'
         annotations:
             foo: bar
`

var parseTests = []struct {
	about       string
	data        string
	expectedBD  *charm.BundleData
	expectedErr string
}{{
	about: "mediawiki",
	data:  mediawikiBundle,
	expectedBD: &charm.BundleData{
		Series: "precise",
		Services: map[string]*charm.ServiceSpec{
			"mediawiki": {
				Charm:    "cs:precise/mediawiki-10",
				NumUnits: 1,
				Options: map[string]interface{}{
					"debug": false,
					"name":  "Please set name of wiki",
					"skin":  "vector",
				},
				Annotations: map[string]string{
					"gui-x": "609",
					"gui-y": "-15",
				},
			},
			"mysql": {
				Charm:    "cs:precise/mysql-28",
				NumUnits: 2,
				To:       []string{"0", "mediawiki/0"},
				Options: map[string]interface{}{
					"binlog-format": "MIXED",
					"block-size":    5,
					"dataset-size":  "80%",
					"flavor":        "distro",
					"ha-bindiface":  "eth0",
					"ha-mcastport":  5411,
				},
				Annotations: map[string]string{
					"gui-x": "610",
					"gui-y": "255",
				},
				Constraints: "mem=8g",
			},
		},
		Machines: map[string]*charm.MachineSpec{
			"0": {
				Constraints: "arch=amd64 mem=4g",
				Annotations: map[string]string{
					"foo": "bar",
				},
			},
		},
		Relations: [][]string{
			{"mediawiki:db", "mysql:db"},
			{"mysql:foo", "mediawiki:bar"},
		},
	},
}, {
	about: "relations specified with hyphens",
	data: `
relations:
    - - "mediawiki:db"
      - "mysql:db"
    - - "mysql:foo"
      - "mediawiki:bar"
`,
	expectedBD: &charm.BundleData{
		Relations: [][]string{
			{"mediawiki:db", "mysql:db"},
			{"mysql:foo", "mediawiki:bar"},
		},
	},
}}

func (*bundleDataSuite) TestParse(c *gc.C) {
	for i, test := range parseTests {
		c.Logf("test %d: %s", i, test.about)
		bd, err := charm.ReadBundleData(strings.NewReader(test.data))
		if test.expectedErr != "" {
			c.Assert(err, gc.ErrorMatches, test.expectedErr)
			continue
		}
		c.Assert(err, gc.IsNil)
		c.Assert(bd, jc.DeepEquals, test.expectedBD)
	}
}

var verifyErrorsTests = []struct {
	about  string
	data   string
	errors []string
}{{
	about: "as many errors as possible",
	data: `
series: "9wrong"

machines:
    0:
         constraints: 'bad constraints'
         annotations:
             foo: bar
    bogus:
    3:
services: 
    mediawiki: 
        charm: "bogus:precise/mediawiki-10"
        num_units: -4
        options: 
            debug: false
            name: Please set name of wiki
            skin: vector
        annotations: 
            "gui-x": 609
            "gui-y": -15
    mysql: 
        charm: "cs:precise/mysql-28"
        num_units: 2
        to: [0, mediawiki/0, nowhere/3, 2, "bad placement"]
        options: 
            "binlog-format": MIXED
            "block-size": 5
            "dataset-size": "80%"
            flavor: distro
            "ha-bindiface": eth0
            "ha-mcastport": 5411
        annotations: 
            "gui-x": 610
            "gui-y": 255
        constraints: "bad constraints"
relations:
    - ["mediawiki:db", "mysql:db"]
    - ["mysql:foo", "mediawiki:bar"]
    - ["arble:bar"]
    - ["arble:bar", "mediawiki:db"]
    - ["mysql:foo", "mysql:bar"]
    - ["mysql:db", "mediawiki:db"]
    - ["mediawiki/db", "mysql:db"]

`,
	errors: []string{
		`bundle declares an invalid series "9wrong"`,
		`machine "3" is not referred to by a placement directive`,
		`machine "bogus" is not referred to by a placement directive`,
		`invalid machine id "bogus" found in machines`,
		`invalid constraints "bad constraints" in machine "0": bad constraint`,
		`invalid charm URL in service "mediawiki": charm URL has invalid schema: "bogus:precise/mediawiki-10"`,
		`invalid constraints "bad constraints" in service "mysql": bad constraint`,
		`negative number of units specified on service "mediawiki"`,
		`too many units specified in unit placement for service "mysql"`,
		`placement "nowhere/3" refers to a service not defined in this bundle`,
		`placement "mediawiki/0" specifies a unit greater than the -4 unit(s) started by the target service`,
		`placement "2" refers to a machine not defined in this bundle`,
		`relation ["arble:bar"] has 1 endpoint(s), not 2`,
		`relation ["arble:bar" "mediawiki:db"] refers to service "arble" not defined in this bundle`,
		`relation ["mysql:foo" "mysql:bar"] relates a service to itself`,
		`relation ["mysql:db" "mediawiki:db"] is defined more than once`,
		`invalid placement syntax "bad placement"`,
		`invalid relation syntax "mediawiki/db"`,
	},
}, {
	about: "mediawiki should be ok",
	data:  mediawikiBundle,
}}

func (*bundleDataSuite) TestVerifyErrors(c *gc.C) {
	for i, test := range verifyErrorsTests {
		c.Logf("test %d: %s", i, test.about)
		bd, err := charm.ReadBundleData(strings.NewReader(test.data))
		c.Assert(err, gc.IsNil)
		err = bd.Verify(func(c string) error {
			if c == "bad constraints" {
				return fmt.Errorf("bad constraint")
			}
			return nil
		})
		if len(test.errors) == 0 {
			c.Assert(err, gc.IsNil)
			continue
		}
		c.Assert(err, gc.FitsTypeOf, (*charm.VerificationError)(nil))
		errors := err.(*charm.VerificationError).Errors
		errStrings := make([]string, len(errors))
		for i, err := range errors {
			errStrings[i] = err.Error()
		}
		sort.Strings(errStrings)
		sort.Strings(test.errors)
		c.Assert(errStrings, jc.DeepEquals, test.errors)
	}
}

func (*bundleDataSuite) TestVerifyCharmURL(c *gc.C) {
	bd, err := charm.ReadBundleData(strings.NewReader(mediawikiBundle))
	c.Assert(err, gc.IsNil)
	for _, u := range []string{
		"wordpress",
		"cs:wordpress",
		"cs:precise/wordpress",
		"precise/wordpress",
		"precise/wordpress-2",
		"local:foo",
		"local:foo-45",
	} {
		bd.Services["mediawiki"].Charm = u
		err := bd.Verify(func(string) error { return nil })
		c.Assert(err, gc.IsNil, gc.Commentf("charm url %q", u))
	}
}

var parsePlacementTests = []struct {
	placement string
	expect    *charm.UnitPlacement
	expectErr string
}{{
	placement: "lxc:service/0",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Service:       "service",
		Unit:          0,
	},
}, {
	placement: "lxc:service",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Service:       "service",
		Unit:          -1,
	},
}, {
	placement: "lxc:99",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Machine:       "99",
		Unit:          -1,
	},
}, {
	placement: "lxc:new",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Machine:       "new",
		Unit:          -1,
	},
}, {
	placement: "service/0",
	expect: &charm.UnitPlacement{
		Service: "service",
		Unit:    0,
	},
}, {
	placement: "service",
	expect: &charm.UnitPlacement{
		Service: "service",
		Unit:    -1,
	},
}, {
	placement: "service45",
	expect: &charm.UnitPlacement{
		Service: "service45",
		Unit:    -1,
	},
}, {
	placement: "99",
	expect: &charm.UnitPlacement{
		Machine: "99",
		Unit:    -1,
	},
}, {
	placement: "new",
	expect: &charm.UnitPlacement{
		Machine: "new",
		Unit:    -1,
	},
}, {
	placement: ":0",
	expectErr: `invalid placement syntax ":0"`,
}, {
	placement: "05",
	expectErr: `invalid placement syntax "05"`,
}, {
	placement: "new/2",
	expectErr: `invalid placement syntax "new/2"`,
}}

func (*bundleDataSuite) TestParsePlacement(c *gc.C) {
	for i, test := range parsePlacementTests {
		c.Logf("test %d: %q", i, test.placement)
		up, err := charm.ParsePlacement(test.placement)
		if test.expectErr != "" {
			c.Assert(err, gc.ErrorMatches, test.expectErr)
		} else {
			c.Assert(err, gc.IsNil)
			c.Assert(up, jc.DeepEquals, test.expect)
		}
	}
}