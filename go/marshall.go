package main
import (
    "encoding/json"
    "fmt"
    "reflect"
)
func main() {
    // marsall int
    i := 100
    marshal_int, _ := json.Marshal(i)
    //check type
    fmt.Println("Before cast: ", reflect.TypeOf(marshal_int))
    fmt.Println("After cast: ", reflect.TypeOf(string(marshal_int)))
    fmt.Println(string(marshal_int))
    // marshall struct
    type Employee struct {
	Name  string
	Age	int	
	Address string
}
	emp := Employee{Name: "George Smith", Age: 30, Address: "Newyork, USA"}
	empData, err := json.Marshal(emp)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(emp.Name)
	fmt.Println(string(empData))
	fmt.Println("Before cast: ", reflect.TypeOf(empData))
	fmt.Println("After cast: ", reflect.TypeOf(string(empData)))
	// Unmarshall into struct
	type Response struct {
	    Name string `json:"name"`
	    Age int `json:"age"`
	    Address string `json:"address"`
	}
	empJsonData := `{"Name":"George Smith","Age":30,"Address":"Newyork, USA"}`	
	empBytes := []byte(empJsonData)
	var resp Response
	json.Unmarshal(empBytes, &resp)
	fmt.Println(resp.Name)
	fmt.Println(resp.Age)
	fmt.Println(resp.Address)
}
