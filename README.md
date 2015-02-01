# Go bindings for Augeas

This package provides Go bindings for [Augeas](http://augeas.net/),
the configuration editing tool.

## Installation

```sh
go get honnef.co/go/augeas
```

## Documentation

Documentation can be found at
[godoc.org](http://godoc.org/honnef.co/go/augeas).

## Examples
### Simple example

```go
package main

import (
	"honnef.co/go/augeas"

	"fmt"
)

func main() {
	ag, err := augeas.New("/", "", augeas.None)
	if err != nil {
		panic(err)
	}

	// There is also Augeas.Version(), but we're demonstrating Get
	// here.
	version, err := ag.Get("/augeas/version")
	fmt.Println(version, err)
}
```

### Extended example

An extended example that fetches all host entries from /etc/hosts can
be found [in the playground](http://play.golang.org/p/aDjm4RWBvP).

## Caveats

The bindings use cgo, so cross-compiling is not possible.
