# planetscale-go

Go package to access the PlanetScale API.


## Install

```bash
go get github.com/planetscale/planetscale-go
```

## Usage

Here is an example usage of the PlanetScale Go client. Please make sure to
handler errors in your production application.


```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/planetscale/planetscale-go"
)

func main() {
	token := os.Getenv("PLANETSCALE_TOKEN")

	ctx := context.Background()

	// create a new PlanetScale API client with the given access token
	client, _ := planetscale.NewClient(
		planetscale.WithAccessToken(token),
	)

	// create a new database
	_, err := client.Databases.Create(ctx, "my-org", &planetscale.CreateDatabaseRequest{
		Database: &planetscale.Database{
			Name:  "my-awesome-database",
			Notes: "This is a test DB created via the planetscale-go API library",
		},
	})

	// list all databases for the given organization
	databases, _ := client.Databases.List(ctx, "my-org")
	fmt.Printf("Found %d databases\n", len(databases))
	for _, db := range databases {
		fmt.Printf("Name: %q\n", db.Name)
		fmt.Printf("Notes: %q\n", db.Notes)
	}

	// delete a database
	_, _ = client.Databases.Delete(ctx, "my-org", "my-awesome-database")
}
```


## Use a custom HTTP Client

You can use a custom HTTP Client with the `planetscale.WithHTTPClient()` option
function:

```go
httpClient := &http.Client{
	Timeout: 15 * time.Second,
}

// create a new PlanetScale API client with the given access token and
// custom HTTP Client
client, _ := planetscale.NewClient(
	planetscale.WithHTTPClient(httpClient),
	planetscale.WithAccessToken(token),
)
```

