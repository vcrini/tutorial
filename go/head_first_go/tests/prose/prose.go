package prose

import "strings"

func JoinWithCommas(phrases []string) string {
  l :=len(phrases)
  result :=strings.Join(phrases[:l-1], ", ")
  result += " e "
  result += phrases[l-1]
  return result
}
