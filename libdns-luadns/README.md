# Lua DNS for `libdns`

[![Go Reference](https://pkg.go.dev/badge/github.com/libdns/luadns.svg)](https://pkg.go.dev/github.com/libdns/luadns)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Lua DNS](https://www.luadns.com/), allowing you to manage DNS records programmatically.

## Authenticating

To authenticate with the Lua DNS API, you need:

1. Your account email address
2. An API key from https://www.luadns.com/api_keys

API keys can be configured as global (access to all zones) or zone-restricted for additional security.

## Example

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/libdns/libdns"
	"github.com/libdns/luadns"
)

func main() {
	provider := &luadns.Provider{
		Email:  "your@email.com",
		APIKey: "your-api-key",
	}

	zone := "example.com."

	// Create a new record
	records, err := provider.AppendRecords(context.Background(), zone, []libdns.Record{
		{
			Type:  "A",
			Name:  "test",
			Value: "192.0.2.1",
			TTL:   time.Hour,
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Created records: %+v\n", records)
}
```

## Features

- ✅ Implements all libdns interfaces (Get, Append, Set, Delete)
- ✅ HTTP Basic Authentication
- ✅ Zone ID caching to minimize API calls
- ✅ Proper error handling
- ✅ Context support for cancellation
- ✅ Rate limit awareness (1200 requests per 5 minutes)

## API Reference

The Lua DNS API documentation is available at: https://www.luadns.com/api.html

## License

MIT License - see LICENSE file for details.
