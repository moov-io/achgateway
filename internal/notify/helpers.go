package notify

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

func convertDollar(in int) string {
	printer := message.NewPrinter(language.Und)
	formatter := number.Decimal(float64(in)/100.0, number.MinFractionDigits(2))
	return printer.Sprint(formatter)
}
