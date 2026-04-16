//go:build tools

package mobile

// Keep golang.org/x/mobile in go.mod for gomobile bind.
import (
	_ "golang.org/x/mobile/bind"
)
