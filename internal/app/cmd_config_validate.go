package app

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

const configValidateUsage = `Usage: msh config validate [flags]

Loads the configuration file and the inventory, validates both, and
prints a summary. Exits 0 when everything is valid, 3 when the
configuration or inventory is invalid, and 8 when a file cannot be read.
`

// validationSummary is the JSON shape of "config validate".
type validationSummary struct {
	Valid      bool     `json:"valid"`
	ConfigFile string   `json:"config_file"`
	Inventory  string   `json:"inventory"`
	Servers    int      `json:"servers"`
	Groups     []string `json:"groups"`
}

func runConfigValidate(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	fs := c.newFlagSet("config validate", configValidateUsage)
	opts.register(fs)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	st, code := c.load(&opts, true)
	if st == nil {
		return code
	}
	if c.rejectCSV(&opts, "config validate") {
		return exitcode.Usage
	}

	groups := map[string]struct{}{}
	for _, s := range st.inv.Servers {
		for _, g := range s.Groups {
			groups[g] = struct{}{}
		}
	}
	sum := validationSummary{
		Valid:      true,
		ConfigFile: st.configFile,
		Inventory:  st.settings.Inventory,
		Servers:    len(st.inv.Servers),
		// maps.Keys returns an iterator; slices.Sorted collects and sorts
		// it, because map iteration order is deliberately random.
		Groups: slices.Sorted(maps.Keys(groups)),
	}

	var err error
	if opts.format != output.Text {
		err = output.WriteObject(c.stdout, opts.format, sum)
	} else {
		configFile := sum.ConfigFile
		if configFile == "" {
			configFile = "(none)"
		}
		err = output.KeyValues(c.stdout, []output.Pair{
			{Key: "Configuration", Value: "valid"},
			{Key: "Config file", Value: configFile},
			{Key: "Inventory", Value: sum.Inventory},
			{Key: "Servers", Value: strconv.Itoa(sum.Servers)},
			{Key: "Groups", Value: strings.Join(sum.Groups, ", ")},
		})
	}
	if err != nil {
		c.errorf("writing output: %v", err)
		return exitcode.IO
	}
	return exitcode.OK
}
