package main

import (
	"fmt"

	"github.com/paketo-buildpacks/go-build/integration/testdata/work_use/find"
)

func main() {
	pattern := "buildpacks"
	data := []string{"paketo", "buildpacks"}
	fmt.Printf("found: %d", len(find.Fuzzy(pattern, data...)))
}
