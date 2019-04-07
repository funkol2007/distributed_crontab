package common


const (
	JOB_KEY_PREFIX      = "/cron/jobs/"
	JOB_KILL_PREFIX     = "/cron/kill/"
	JOB_LOCK_PREFIX     = "/cron/lock/"
	JOB_RGEISTER_PREFIX = "/cron/workers/"
	JOB                 = "job"
	JOB_NAME            = "name"
	JOB_SKIP            = "skip"
	JOB_LIMIT           = "limit"
	MAX_NUM_JOB_QUEUE   = 1000
	MAX_NUM_LOG_QUEUE   = 1000
	JOB_EVENT_SAVE      = 0
	JOB_EVENT_DELETE    = 1
	JOB_EVENT_KILL      = 2
)