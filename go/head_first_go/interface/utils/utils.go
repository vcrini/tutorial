package utils

import (
	"fmt"
	"log"
)

type TapePlayer struct {
	Batteries string
}

func (t TapePlayer) Play(song string) {
	fmt.Println("Suonando ", song)
}
func (t TapePlayer) Stop() {
	fmt.Println("Fermo!")
}

type TapeRecorder struct {
	Microphones int
}

func (t TapeRecorder) Play(song string) {
	fmt.Println("Suonando ", song)
}
func (t TapeRecorder) Recording(song string) {
	fmt.Println("Registrando", song)
}
func (t TapeRecorder) Stop() {
	fmt.Println("Fermo!")
}

type Device interface {
	Play(string)
	Stop()
}

func TryOut(d Device) {
	d.Play("Test track")
	d.Stop()
	recorder, ok := d.(TapeRecorder)
	if ok {
		recorder.Recording("Test record")
	} else {
		log.Fatal("it's not taperecorder!!!11!")

	}
}
