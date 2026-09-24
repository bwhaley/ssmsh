package commands

const lsUsage = "ls [-rR] [region:]path ..."

func ls(c *Context) error {
	paths, recursive := checkRecursion(c.Args)
	if len(paths) == 0 {
		paths = []string{ps.Cwd}
	}
	all := []string{}
	for _, name := range paths {
		p, err := parsePath(name)
		if err != nil {
			return err
		}
		names, err := ps.List(commandContext, p, recursive)
		if err != nil {
			return err
		}
		if len(paths) > 1 && cfg.Default.Output != "json" {
			shell.Println(name + ":")
		}
		if cfg.Default.Output != "json" {
			for _, v := range names {
				shell.Println(v)
			}
		}
		all = append(all, names...)
	}
	if cfg.Default.Output == "json" {
		return printJSON(all)
	}
	return nil
}
