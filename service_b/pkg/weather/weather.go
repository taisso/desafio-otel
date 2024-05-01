package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

type WeatherApi struct {
	apiKey string
	trace  trace.Tracer
}

func NewWeather(apiKey string, tracer trace.Tracer) *WeatherApi {
	return &WeatherApi{apiKey: apiKey, trace: tracer}
}

func (w *WeatherApi) GetWeather(ctx context.Context, lat, lon string) (*Weather, error) {
	ctx, span := w.trace.Start(ctx, "service-b:get-temp")
	defer span.End()

	weatherUrl := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s,%s", w.apiKey, lat, lon)
	weather, err := fetch[Weather](ctx, weatherUrl)

	if err != nil {
		return nil, err
	}

	return weather, nil
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
