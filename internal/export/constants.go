package export

// Cron field indices for 5-field cron expressions.
const (
	cronFieldMinute = 0
	cronFieldHour   = 1
	cronFieldDOM    = 2
	cronFieldMonth  = 3
	cronFieldDOW    = 4
	cronFieldCount  = 5
)

// Supported export format names.
const (
	FormatCrontab = "crontab"
	FormatLaunchd = "launchd"
	FormatSystemd = "systemd"
)
