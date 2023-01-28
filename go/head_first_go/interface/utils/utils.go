package utils
import "fmt"
type TapePlayer struct  {
  Batteries string
}
func (t TapePlayer) Play(song string) {
  fmt.Println("Suonando ", song)
}
func (t TapePlayer) Stop() {
  fmt.Println("Fermo!")
}

type TapeRecorder struct  {
  Microphones int
}
func (t TapeRecorder) Play(song string) {
  fmt.Println("Suonando ", song)
}
func (t TapeRecorder) Recording(song string) {
  fmt.Println("Registrando")
}
func (t TapeRecorder) Stop() {
  fmt.Println("Fermo!")
}
type Device interface {
  Play(string)
  Stop()
}
