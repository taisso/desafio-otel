### Desafio OTEL (OpenTelemetry) – Demo com dois serviços em Go

Este repositório demonstra instrumentação de tracing distribuído com OpenTelemetry em dois serviços HTTP escritos em Go, integrados ao OpenTelemetry Collector e visualização no Zipkin.

- **service_a (porta 8081)**: recebe uma requisição com `cep` e chama o `service_b` propagando o contexto de trace.
- **service_b (porta 8080)**: resolve o CEP para coordenadas (Nominatim) e busca a temperatura atual (WeatherAPI), retornando cidade e temperaturas em C/F/K.

### Arquitetura
- **Zipkin (9411)**: visualização dos traces.
- **OTEL Collector (4317)**: recebe spans via OTLP/gRPC dos serviços.
- **service_a** → chama → **service_b** → (Nominatim + WeatherAPI)

### Requisitos
- Docker e Docker Compose
- Chave de API da WeatherAPI para o `service_b` (`https://www.weatherapi.com/`)

### Variáveis de ambiente
Crie um arquivo `.env` no diretório `service_b/` antes de subir os containers (o Dockerfile copia esse arquivo para a imagem):

```
WEATHER_API_KEY=SEU_TOKEN_AQUI
```

### Como iniciar
1. Certifique-se de ter criado `service_b/.env` com a variável acima.
2. Suba toda a stack com build:

```
docker compose up --build
```

Isso iniciará: `zipkin` (9411), `otel-collector` (4317), `service_a` (8081) e `service_b` (8080).

### Portas
- service_a: 8081
- service_b: 8080
- zipkin: 9411
- otel-collector (OTLP/gRPC): 4317

### Endpoints e exemplos
- **service_a** `POST /` (localhost:8081)
  - Body: `{ "cep": "18085090" }` (8 dígitos)

```
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"cep":"18085090"}' \
  http://localhost:8081/
```

- **service_b** `GET /{cep}` (localhost:8080)

```
curl http://localhost:8080/18085090
```

Você também pode usar o arquivo `request.http` na raiz do projeto.

### Observabilidade (Zipkin)
- Acesse `http://localhost:9411` no navegador.
- Busque por serviços: "service-a" ou "service-b" para visualizar os traces.

![Exemplo no Zipkin](https://i.ibb.co/55LgGJz/2024-05-01-12-10.png)

### Estrutura do repositório (resumo)
```
.
├── docker-compose.yml
├── request.http
├── service_a/
│   ├── Dockerfile
│   └── main.go
└── service_b/
    ├── Dockerfile
    ├── main.go
    └── pkg/
        ├── nominatim/
        │   ├── model.go
        │   └── nominatim.go
        └── weather/
            ├── model.go
            └── weather.go
```

### Erros comuns
- 422/400 para CEP inválido: use exatamente 8 dígitos.
- 404 em `service_b`: CEP não encontrado pelo Nominatim.
- 500: verifique conectividade externa e se `WEATHER_API_KEY` é válido.
- Traces não aparecem: confirme que o `otel-collector` está no ar e que os serviços conseguiram conectar em `otel-collector:4317`.

### Desenvolvimento local (opcional)
Os serviços são Go puros, então você pode rodá-los localmente (fora do Docker). Para manter a telemetria funcionando, a configuração aponta para `otel-collector:4317` (nome de container). Em execução local, ajuste o endpoint do OTLP ou suba tudo via Docker (recomendado para este demo).