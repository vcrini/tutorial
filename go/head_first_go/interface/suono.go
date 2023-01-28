package main
import "example.com/go/utils/utils"


func playList(device utils.Device, songs []string) {
  for _, song := range songs {
    device.Play(song)
  }
}

func main() {
  player := utils.TapePlayer{}
  mixtape := []string{"Inno del corpo sciolto", "Tapparelle"}
  playList(player, mixtape)
  player2 := utils.TapeRecorder{}
  playList(player2, mixtape)
}
