package main

import  "fmt" 
        
func main() {
  var quantity int
  var length, width float64
  var customerName string
  quantity = 4
  length, width = 1.2, 2.5
  customerName = "Damon Cole"

  fmt.Println(customerName)
  fmt.Println("Ha ordinato ", quantity, " fogli")
  fmt.Println("Ogniuno con l'area pari a ")
  fmt.Println(length*width, "metri quadri")
}
