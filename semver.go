//go:generate sh -c "printf %s $(git-semver) > commit.txt"
package main

import _ "embed"

//go:embed commit.txt
var Semver string
