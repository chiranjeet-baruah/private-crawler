package utils

import (
	"fmt"
	cutils "github.com/Semantics3/sem3-go-crawl-utils/utils"
	"regexp"
	"strconv"
)

var includesSpacesRegex *regexp.Regexp

func CleanupUpc(upc string) string {
	includesNonDigits, _ := cutils.ApplyRegex(upc, upc, "includesNonDigitsRegex", `[^\d]`, "")

	if includesNonDigits {
		fmt.Printf("CLEANUP_UPC: upc (%s) seems invalid because it has non-digits\n", upc)
		return upc
	}

	upcLikeEAN13, _ := cutils.ApplyRegex(upc, upc, "likeEAN13Regex", `^0(\d{12})$`, "")
	upcLikeGTIN14, _ := cutils.ApplyRegex(upc, upc, "likeGTIN14Regex", `^00(\d{12})$`, "")

	upc = removeAllSpaces(upc)
	if upcLikeEAN13 {
		return upc
	} else if upcLikeGTIN14 {
		return upc
	} else if len(upc) <= 11 {
		lastDigitAsString := fmt.Sprintf("%c", upc[len(upc)-1])
		lastDigit, _ := strconv.Atoi(lastDigitAsString)
		upcWithoutLastDigit := upc[0 : len(upc)-1]
		checkDigit := computeCheckDigit(padUpc(upcWithoutLastDigit))
		if checkDigit != lastDigit {
			upc = fmt.Sprintf("%s%d", upc, checkDigit)
		} else {
			fmt.Printf("last digit (%d) is the same as the check digit (%d)\n", lastDigit, checkDigit)
		}
		upc = padUpc(upc)
	}

	return upc
}

func removeAllSpaces(value string) string {
	if includesSpacesRegex == nil {
		includesSpacesRegex, _ = regexp.Compile(`\s*`)
	}
	return includesSpacesRegex.ReplaceAllString(value, "")
}

func padUpc(upc string) string {
	for len(upc) < 12 {
		upc = fmt.Sprintf("0%s", upc)
	}
	return upc
}

func computeCheckDigit(upc string) int {
	evenSum := 0
	oddSum := 0
	for i := 1; i <= len(upc); i++ {
		number, _ := strconv.Atoi(fmt.Sprintf("%c", upc[i-1]))
		if i%2 == 0 {
			evenSum += number
		} else {
			oddSum += number
		}
	}
	oddSum *= 3
	checkDigit := (10 - (oddSum+evenSum)%10) % 10
	return checkDigit
}
