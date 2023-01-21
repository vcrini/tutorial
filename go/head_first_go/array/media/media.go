package media

func Media(elements []float64) float64 {
	sum := 0.0
	for _, v := range elements {
		sum += v
	}
	n := float64(len(elements))
	return sum / n
}
