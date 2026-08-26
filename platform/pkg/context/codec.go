// cmdr: cmdr: context\codec.go

package context

import (
	"encoding/json"
	"errors"
)

//#region Encode
func Encode(ctx RequestContext) ([]byte, error) {
	return json.Marshal(ctx)
}

//#region Decode
func Decode(data []byte) (*RequestContext, error) {
	var ctx RequestContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, err
	}
	return &ctx, nil
}

var ErrInvalidContext = errors.New("invalid request context")
