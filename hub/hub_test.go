package hub

import (
	"fmt"
	"testing"
)

func TestHandler(t *testing.T) {
	a := [][]any{[]any{1}}
	do(a)
}
func do(p [][]any) {

	fmt.Println(p)
}
