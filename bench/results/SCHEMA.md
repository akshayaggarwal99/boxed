# CSV schemas

## coldstart.csv
`i,create_ms,first_exec_ms,destroy_ms,total_ms`

## throughput.csv
`conc,n,completed,errors,elapsed_s,rps` — appended once per concurrency level

## overhead.csv
`i,rss_mib,cpu_pct` — sampled 5s after sandbox creation

## agent_trace.csv
`task,llm_ms,create_ms,exec_ms,destroy_ms,total_ms,passed`
`passed` ∈ {true,false,ERR}

## escapes.csv
`test,result` — `result` ∈ {DENIED, SUCCEEDED, ERROR}
