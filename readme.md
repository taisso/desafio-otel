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

### Para visualizar no Zipkin

Para visualizar no zipkin busque no campo de pesquisa por algum dos serviços
que são: "service-a" ou "service-b"
![alt text](https://i.ibb.co/55LgGJz/2024-05-01-12-10.png)