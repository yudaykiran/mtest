> We shall list down the code, design, coverage, functional test, etc based
challenges, issues, etc & their corresponding resolutions. It is all related to
programming. Some of these may exist, while some would have been taken care of. 
Some of these may have been reported at github issues. Neverthless, we shall 
list specific cases for our new contributors to appreciate the effort & get them
to the programming level enjoyed by the core / regular contributors.

#### server gave HTTP response to HTTPS client

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
  
#### Log message not readable

```bash
INFO:  [%!q(*mtest.Report=<nil>) %!q(*mtest.Report=&{mserver.runner mserver.volume.remove.usecase 0xc420011d60 FAILED false})]
```
