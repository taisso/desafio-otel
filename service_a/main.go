package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

var (
	ErrStatusInternalServer = errors.New("internal error")
	ErrInvalidZipCode       = errors.New("invalid zipcode")
)

type BodyRequest struct {
	Cep string `json:"cep"`
}

type ResponseError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func InitProvider() (func(context.Context) error, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("service-a"),
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

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shutdown, err := InitProvider()
	if err != nil {
		log.Fatal(err)
	}

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

	router.Post("/", Handler)

	http.ListenAndServe(":8081", router)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("service-tracer-a")
	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	ctx, span := tracer.Start(ctx, "get-service-b")
	defer span.End()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		ToError(w, http.StatusInternalServerError, ErrStatusInternalServer)
		return
	}

	var bodyRequest BodyRequest
	if err := json.Unmarshal(body, &bodyRequest); err != nil {
		log.Println(err)
		ToError(w, http.StatusInternalServerError, ErrStatusInternalServer)
		return
	}

	kind := reflect.TypeOf(bodyRequest.Cep).Kind()
	if len(bodyRequest.Cep) != 8 || kind != reflect.String {
		ToError(w, http.StatusBadRequest, ErrInvalidZipCode)
		return
	}

	url := fmt.Sprintf("http://service_b:8080/%s", bodyRequest.Cep)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Println(err)
		ToError(w, http.StatusInternalServerError, ErrStatusInternalServer)
		return
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		ToError(w, http.StatusInternalServerError, ErrStatusInternalServer)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		ToError(w, http.StatusInternalServerError, ErrStatusInternalServer)
		return
	}

	w.Write(data)
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
