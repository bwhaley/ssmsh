package commands

const mvUsage = "mv [--dry-run] [key=DESTINATION_KEY] src dest"

func mv(c *Context) error { return copyOrMove(c, true) }
