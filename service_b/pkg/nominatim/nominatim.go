package nominatim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

var (
	ErrNotFound = errors.New("empty")
)

type NominatimApi struct {
	tracer trace.Tracer
}

func NewNominatim(tracer trace.Tracer) *NominatimApi {
	return &NominatimApi{
		tracer: tracer,
	}
}

func (n *NominatimApi) GetLocation(ctx context.Context, cep string) (*Nominatim, error) {
	ctx, span := n.tracer.Start(ctx, "get-cep")
	defer span.End()

	nominatimUrl := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=jsonv2&addressdetails=1", cep)
	nominatims, err := fetch[[]Nominatim](ctx, nominatimUrl)

	if err != nil {
		return nil, err
	}

	if len(*nominatims) == 0 {
		return nil, ErrNotFound
	}

	nominatim := (*nominatims)[0]

	return &nominatim, nil
}

func fetch[T any](ctx context.Context, url string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var data T
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
