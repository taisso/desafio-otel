### Iniciar

```
docker compose up --build
```

### Portas
- service_a: 8081
- service_b: 8080
- zikpin: 9411

### Endpoint
```
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"cep":"18085090"}' \
  http://localhost:8081
```
>> Também tem o arquivo request.http caso queira