fields @timestamp, @message
| filter @message like /REPORT RequestId/
| stats count(*) as totalInvocations


fields @timestamp, @message
| filter @message like /Task timed out/
| stats count(*) as timeouts


fields @timestamp, @message
| filter @message like /INIT_START/
| stats count(*) as coldStarts


fields @timestamp, @message
| filter @message like /REPORT RequestId/
| parse @message /Duration: (?<duration>[0-9.]+) ms.*Max Memory Used: (?<maxMem>\d+) MB/
| stats
    avg(duration) as avgDurationMs,
    min(duration) as minDurationMs,
    max(duration) as maxDurationMs,
    avg(maxMem) as avgMemoryMB,
    min(maxMem) as minMemoryMB,
    max(maxMem) as maxMemoryMB

