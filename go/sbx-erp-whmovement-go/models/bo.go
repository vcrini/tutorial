// Package models describes models
package models

import (
	"encoding/xml"
	"time"
)

type Element struct {
	Content   string `xml:"-"`
	SocCod    string `xml:"socCod"`
	BoType    string `xml:"boType"`
	BoCod     string `xml:"boCod"`
	Operation string `xml:"operation"` // Sync/Delete
	Size      int64
	Key       string
}

type TBMGT struct { // Da XSD
	XMLName xml.Name `xml:"TBMGT"`
	ID      string   `xml:"id"`
	// ... 20+ campi da XSD whmovement
}

type Block struct {
	UUID     string
	Create   time.Time
	Elements []Element
	Errors   []string
}

type DecodeError struct {
	SocCod    string
	BoType    string
	BoCod     string
	Operation string
}
