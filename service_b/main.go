package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/taisso/service_b/pkg/nominatim"
	"github.com/taisso/service_b/pkg/weather"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type Response struct {
	City  string  `json:"city"`
	TempC float32 `json:"temp_c"`
	TempF float32 `json:"temp_f"`
	TempK float32 `json:"temp_k"`
}

type ResponseError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

var (
	ErrInvalidZipCode       = errors.New("invalid zipcode")
	ErrStatusInternalServer = errors.New("internal error")
	ErrNotFoundZipCode      = errors.New("can not find zipcode")
)

type CepTemperature struct {
	weather   weather.IWeather
	nominatim nominatim.INominatim
	tracer    trace.Tracer
}

func NewCepTemperature(weather weather.IWeather, nomination nominatim.INominatim, tracer trace.Tracer) *CepTemperature {
	return &CepTemperature{weather: weather, nominatim: nomination, tracer: tracer}
}

func (ct CepTemperature) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ct.GetCepTemperatureHandler(w, r)
}

func (ct CepTemperature) GetCepTemperatureHandler(w http.ResponseWriter, r *http.Request) {
	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	cep := r.PathValue("cep")

	if len(cep) != 8 {
		ToError(w, http.StatusUnprocessableEntity, ErrInvalidZipCode)
		return
	}

	nominatimResponse, err := ct.nominatim.GetLocation(ctx, cep)
	if err != nil {
		if err == nominatim.ErrNotFound {
			ToError(w, http.StatusNotFound, ErrNotFoundZipCode)
			return
		}
		log.Println(err)
		ToError(w, http.StatusInternalServerError, ErrStatusInternalServer)
		return
	}

	lat := nominatimResponse.Lat
	lng := nominatimResponse.Lon

	weatherResponse, err := ct.weather.GetWeather(ctx, lat, lng)
	if err != nil {
		log.Println(err)
		ToError(w, http.StatusInternalServerError, ErrStatusInternalServer)
		return
	}

	ToJson(w, http.StatusOK, Response{
		City:  weatherResponse.Location.Name,
		TempC: float32(weatherResponse.Current.TempC),
		TempF: float32(weatherResponse.Current.TempF),
		TempK: float32(weatherResponse.Current.TempC + 273),
	})
}

func InitProvider() (func(context.Context) error, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("service-b"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "otel-collector:4317",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider.Shutdown, nil
}

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file")
	}
}

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shutdown, err := InitProvider()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Inicializado o InitProvider")

	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal("failed to shutdown TracerProvider: %w", err)
		}
	}()

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	apiKey := os.Getenv("WEATHER_API_KEY")
	tracer := otel.Tracer("service-b-tracer")
	cepTemperature := NewCepTemperature(weather.NewWeather(apiKey, tracer), nominatim.NewNominatim(tracer), tracer)
	router.Handle("GET /{cep}", cepTemperature)

	http.ListenAndServe(":8080", router)
}

func ToJson(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		panic(err)
	}
}

func ToError(w http.ResponseWriter, statusCode int, err error) {
	ToJson(
		w,
		statusCode,
		ResponseError{
			Code:    statusCode,
			Message: err.Error(),
		},
	)
}
