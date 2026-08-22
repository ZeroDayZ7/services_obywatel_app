package shared

import (
	"fmt"
	"time"
)

func GenerateAgreementNumber(t time.Time) string {
	return fmt.Sprintf("AGR/%s/%d", t.Format("20060102"), t.Unix()%100000)
}
