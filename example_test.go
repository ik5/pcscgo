package pcscgo

import "fmt"

// ExampleSCardEstablishContext demonstrates context establishment.
func ExampleSCardEstablishContext() {
	var ctx SCardContext
	err := SCardEstablishContext(SCardScopeUser, nil, nil, &ctx)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Context: 0x%x\n", ctx)
	SCardReleaseContext(ctx)
}
