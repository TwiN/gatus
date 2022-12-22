# whois
![test](https://github.com/TwiN/whois/workflows/test/badge.svg?branch=master)

Lightweight library for retrieving WHOIS information on a domain.

It automatically retrieves the appropriate WHOIS server based on the domain's TLD by first querying IANA.


## Usage
### As an executable
To install it:
```console
go install github.com/TwiN/whois/cmd/whois@latest
```
To run it:
```console 
whois example.com
```

### As a library
```console
go get github.com/TwiN/whois
```

#### Query
If all you want is the text a WHOIS server would return you, you can use the `Query` method of the `whois.Client` type:
```go
package main

import "github.com/TwiN/whois"

func main() {
    client := whois.NewClient()
    output, err := client.Query("example.com")
    if err != nil {
    	panic(err)
    }
    println(output)
}
```

#### QueryAndParse
If you want specific pieces of information, you can use the `QueryAndParse` method of the `whois.Client` type:
```go
package main

import "github.com/TwiN/whois"

func main() {
    client := whois.NewClient()
    response, err := client.QueryAndParse("example.com")
    if err != nil {
    	panic(err)
    }
    println(response.ExpirationDate.String()) 
}
```
Note that because there is no standardized format for WHOIS responses, this parsing may not be successful for every single TLD.

Currently, the only fields parsed are:
- `ExpirationDate`: The time.Time at which the domain will expire
- `DomainStatuses`: The statuses that the domain currently has (e.g. `clientTransferProhibited`)
- `NameServers`: The nameservers currently tied to the domain

If you'd like one or more other fields to be parsed, please don't be shy and create an issue or a pull request.

#### Caching referral WHOIS servers
The way that WHOIS scales is by having one "main" WHOIS server, namely `whois.iana.org:43`, refer to other WHOIS server
on a per-TLD basis. 

In other word, let's say that you wanted to have the WHOIS information for `example.com`. 
The first step would be to query `whois.iana.org:43` with `com`, which would return `whois.verisign-grs.com`.
Then, you would query `whois.verisign-grs.com:43` for the WHOIS information on `example.com`.

If you're querying a lot of servers, making two queries instead of one can be a little wasteful, hence `WithReferralCache(true)`:
```go
client := whois.NewClient().WithReferralCache(true)
```
The above will cache the referral WHOIS server for each TLD, so that you can directly query the appropriate WHOIS server
instead of first querying `whois.iana.org:43` for the referral.

