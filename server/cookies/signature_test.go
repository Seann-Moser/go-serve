package cookies

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type AuthTestCase struct {
	Name         string
	Id           string
	Key          string
	expectedSign string
	isValid      bool
	CompareId    string
	CompareKey   string
}

func TestAuthSignature_Valid(t *testing.T) {
	tcs := []AuthTestCase{
		{
			Name:         "Match",
			Id:           "1",
			Key:          "1",
			expectedSign: "9kHtuAgPnMxPwun7BM86ISfEthDajqBQQzDWX5jnLkE=",
			isValid:      true,
			CompareId:    "1",
			CompareKey:   "1",
		},
		{
			Name:         "Different IDs",
			Id:           "1",
			Key:          "1",
			expectedSign: "9kHtuAgPnMxPwun7BM86ISfEthDajqBQQzDWX5jnLkE=",
			isValid:      false,
			CompareId:    "",
			CompareKey:   "1",
		},
		{
			Name:         "Different Signature",
			Id:           "1",
			Key:          "1",
			expectedSign: "9kHtuAgPnMxPwun7BM86ISfEthDajqBQQzDWX5jnLkE",
			isValid:      false,
			CompareId:    "1",
			CompareKey:   "1",
		},
	}
	cookies := New("1234", true, 5*time.Minute, false, nil, zap.L())
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			day, _ := time.Parse("2006-04-02", "2022-10-14")
			firstAuth := cookies.GetAuthSignature(tc.Id, tc.Key, &day, nil)

			if firstAuth.Signature != tc.expectedSign && tc.isValid {
				assert.Same(t, firstAuth.Signature, tc.expectedSign)
			}
			secondAuth := cookies.GetAuthSignature(tc.CompareId, tc.CompareKey, &day, nil)
			if firstAuth.Signature != secondAuth.Signature && tc.isValid {
				assert.Same(t, secondAuth.Signature, firstAuth.Signature)
			}

		})
	}
}
