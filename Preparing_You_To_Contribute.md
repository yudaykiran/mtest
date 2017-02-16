### What is this all about ?

We shall list down the essentials of code, design, coverage, functional testing,
etc. based challenges, issues & their corresponding resolutions or approaches we
have taken. It is all related to programming. Some of these challenges may still 
exist, while some would have been taken care of. Some of these may have been 
reported at github issues. 

Neverthless, we shall list specific cases to enable new contributors to appreciate
the effort & get them to the programming level enjoyed by the core / regular 
contributors.

On the other hand, this ensures seasoned golang programmers to raise issues w.r.t 
design, programming, apporach, or anything they find in-appropriate.

#### ERROR: server gave HTTP response to HTTPS client

- This is the exact error received by mtest while trying to communicate with mayaserver

```bash
ERROR: RequestError: send request failed
caused by: Get https://172.28.128.4:5656/latest/meta-data/instance-id: 
http: server gave HTTP response to HTTPS client
```

- Solution:
  - (Workaround) aws sdk's config property was set

```go
DisableSSL: aws.Bool(true),
```
  
#### LOGGING: Log message was not readable

- This is what we got in our logs that tries to dump an array of mtest reports.

```bash
INFO:  [%!q(*mtest.Report=<nil>) %!q(*mtest.Report=&{mserver.runner mserver.volume.remove.usecase 0xc420011d60 FAILED false})]
```
