//File: service.go

package mystr

import (
"strings"
"github.com/go-kit/kit/log"
)

type Service interface {
IsPal(string) string
Reverse(string) string
}

type myStringService struct {
log log.Logger
}


func NewService() Service {
        return &myStringService{}
}


func (svc *myStringService) IsPal(s string) string {
reverse := svc.Reverse(s)
if strings.ToLower(s) != reverse {
  return "Not palindrome"
}
return "palindrome"
}

func (svc *myStringService) Reverse(s string) string {
rns := []rune(s) // convert to rune
for i, j := 0, len(rns)-1; i < j; i, j = i+1, j-1 {

  // swap the letters of the string,
  // like first with last and so on.
  rns[i], rns[j] = rns[j], rns[i]
}

// return the reversed string.
return strings.ToLower(string(rns))
}
