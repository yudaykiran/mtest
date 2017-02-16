### What is this all about ?

We shall list down the **essentials** of code, design, coverage, issues,
etc. termed as **items** & each of these items' corresponding resolutions, 
approaches etc. core contributors had taken earlier or is currently in place. 

It might boil down to **programming essentials** or **golang programming essentials**.
Some of these items may still exist as issues, while some would have been taken 
care of. Some of these may have been reported as github issues. 

Neverthless, we shall list specific cases that enables **new mtest contributors** 
to appreciate the thought processes & effort that has gone into mtest & get them
to the programming level enjoyed by the core / regular **mtest** contributors.

On the other hand, this ensures seasoned programmers, golang or otherwise to 
talk, point out, or raise issues w.r.t design, code, programming style, approach, 
or anything they find can be done in a better manner.

> **If not this guide**, then you need to have entire time in the world to go through
each & every discussion we had at slack or understand each commit or go through
each github issue. If that sounds possible then you do not need this guide. This 
in turn also signifies the importance of mentioning **date** to each item & its 
approaches.

### Why this name ?

Other names that came to our mind were:

- Challenges_Faced
- FAQs, 
- Curated_Thoughts

However, we wanted folks to understand & enable them to contribute to the project
with little learning curve & hence this name.

### Simple Items

#### LOGGING: Log message was not readable

- This is what we got in our logs that tries to dump an array of mtest reports.
  - What is this memory address `0xc420011d60` ?

```bash
  # dated 16/Feb/2017

  INFO:  [%!q(*mtest.Report=<nil>) 
    %!q(*mtest.Report=&{
      mserver.runner 
      mserver.volume.remove.usecase 
      0xc420011d60 
      FAILED
      false})]
```

- Solution
  - Used error.Error() than error which is a struct that needed to be de-referenced

```bash
  INFO:  [%!s(*mtest.Report=<nil>) %!s(*mtest.Report=&{
    mserver.runner 
    mserver.volume.create.usecase 
    AWS Error:  RequestError send request failed Post 
      https://ec2."any-zone.amazonaws.com/: dial tcp: 
      lookup ec2."any-zone.amazonaws.com: invalid domain name
    FAILED 
    false})]

```

### Intermediate Items

#### ERROR: server gave HTTP response to HTTPS client

- This is the error received by mtest trying to communicate with mayaserver

```bash
  # dated 16/Feb/2017

  ERROR: RequestError: send request failed
  caused by: Get https://172.28.128.4:5656/latest/meta-data/instance-id: 
      http: server gave HTTP response to HTTPS client
```

- Solution as on 16/Feb/2017:
  - mtest uses aws-sdk-go lib. 
  - Its config property was set

```go
  DisableSSL: aws.Bool(true),
```
  
### Advanced Items
