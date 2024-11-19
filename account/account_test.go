package account

import (
	"fmt"

	"testing"
)

func TestCreateAccount(t *testing.T) {

	privateKey, address, err := CreateAccount()
	if err != nil {
		fmt.Printf("ERROR %v", err)
	}
	fmt.Println("개인키 ::", privateKey)
	fmt.Println("주소 ::: ", address)

	if !IsValidAddress(address) {
		fmt.Println("씨발")
		return
	}
	fmt.Println("썽공!!")

}
