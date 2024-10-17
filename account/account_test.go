package account

import (
	"fmt"
	"regexp"

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

func IsValidAddress(address string) bool {
	// 0x 시작 && 총 42자 && 40자 16진수
	// ^ : 시작
	// [0-9a-fA-f] : 이 부분은 괄호 안에 있는 문자들 중 하나를 허용
	// {40} 앞 패턴이 몇번 있어야 하는지
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(address)
}
