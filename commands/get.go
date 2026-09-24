package commands

import (
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

const getUsage = "get [region:]parameter ..."

func get(c *Context) error {
	if len(c.Args) == 0 {
		return fmt.Errorf("usage: %s", getUsage)
	}
	groups := map[string][]string{}
	for _, arg := range c.Args {
		p, err := parsePath(arg)
		if err != nil {
			return err
		}
		groups[p.Region] = append(groups[p.Region], p.Name)
	}
	regions := make([]string, 0, len(groups))
	for r := range groups {
		regions = append(regions, r)
	}
	sort.Strings(regions)
	params := []types.Parameter{}
	for _, r := range regions {
		out, err := ps.Get(commandContext, groups[r], r)
		if err != nil {
			return err
		}
		params = append(params, out...)
	}
	return printParameters(params)
}
