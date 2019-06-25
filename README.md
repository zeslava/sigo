# sigo
Pure Go detector of file type by signatures

**sigo** uses radix-tree for storing files signatures.

Detect reading from input byte-by-byte and find signature in the tree.

### Usage
```go
package main

import (
	"fmt"
	"os"
	
	"github.com/slavablind91/sigo/detector"
)

func main() {
	file, _ := os.Open("filename")
	t, _ := detector.Detect(file)
	fmt.Printf("file type is %s", t.Description)
}

```

#### Resources
Most information is took from here: https://www.garykessler.net/library/file_sigs.html

##### !WARNING!
Project under development at an earlier stage